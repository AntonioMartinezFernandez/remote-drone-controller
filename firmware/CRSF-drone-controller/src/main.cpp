#include <WiFi.h>
#include <WiFiUdp.h>

// ----------------------------------------------------
// Control UDP packet format: "roll,pitch,throttle,yaw,arm"

// Example packets (send with netcat):

// # Disarmed, idle
// echo -n "992,992,172,992,0" | nc -u -w1 192.168.4.1 4210

// # Armed, low throttle
// echo -n "992,992,172,992,1" | nc -u -w1 192.168.4.1 4210

// # Armed, half throttle, centered sticks
// echo -n "992,992,992,992,1" | nc -u -w1 192.168.4.1 4210

// # Continuous 50Hz stream for testing
// while true; do echo -n "992,992,992,992,1" | nc -u -w0 192.168.4.1 4210; sleep 0.02; done

// ----------------------------------------------------

// -------- WIFI AP --------

const char *ssid = "AMF-DRONE";
const char *password = "12345678";

WiFiUDP udp;

const uint16_t UDP_PORT = 4210;

// -------- CRSF UART --------

HardwareSerial CRSFSerial(2);

#define CRSF_TX_PIN 17
#define CRSF_BAUDRATE 420000

// -------- CRSF channel value range (CRSF convention) --------
#define CRSF_MIN 172
#define CRSF_MID 992
#define CRSF_MAX 1811

// -------- Failsafe --------
// If no valid UDP packet is received within this time, throttle is
// forced low and the arm channel is forced to "disarmed".
#define FAILSAFE_TIMEOUT_MS 500

// canales CRSF
uint16_t channels[16];

uint32_t lastPacketMs = 0;
bool failsafeActive = false;

// ----------------------------------------------------
// CRC8 DVB-S2 used by CRSF
// ----------------------------------------------------

uint8_t crc8_dvb_s2(const uint8_t *ptr, uint8_t len)
{
  uint8_t crc = 0;

  while (len--)
  {
    crc ^= *ptr++;

    for (uint8_t i = 0; i < 8; i++)
    {
      if (crc & 0x80)
        crc = (crc << 1) ^ 0xD5;
      else
        crc <<= 1;
    }
  }

  return crc;
}

// ----------------------------------------------------
// Create CRSF RC_CHANNELS_PACKED packet
// ----------------------------------------------------

void sendCRSF()
{
  uint8_t packet[26];

  packet[0] = 0xC8; // flight controller direction
  packet[1] = 24;   // length after this byte (type + payload + crc)
  packet[2] = 0x16; // RC_CHANNELS_PACKED

  uint32_t bitBuffer = 0;
  uint8_t bits = 0;
  uint8_t index = 3;

  for (int ch = 0; ch < 16; ch++)
  {
    // CRSF channels are 11 bits wide (0-2047). Mask defensively so a
    // bad value can never bleed into the neighbouring channel's bits.
    uint16_t value = channels[ch] & 0x07FF;

    bitBuffer |= ((uint32_t)value << bits);
    bits += 11;

    while (bits >= 8)
    {
      packet[index++] = bitBuffer & 0xFF;
      bitBuffer >>= 8;
      bits -= 8;
    }
  }

  packet[25] = crc8_dvb_s2(&packet[2], 23);

  CRSFSerial.write(packet, 26);
}

// ----------------------------------------------------
// Apply a parsed channel value: validate range, clamp to CRSF limits
// ----------------------------------------------------

uint16_t clampChannel(long value)
{
  if (value < CRSF_MIN)
    value = CRSF_MIN;
  if (value > CRSF_MAX)
    value = CRSF_MAX;
  return (uint16_t)value;
}

// ----------------------------------------------------
// Put the aircraft into a safe state: throttle low, sticks centered,
// arm channel forced to "disarmed".
// ----------------------------------------------------

void applyFailsafe()
{
  channels[0] = CRSF_MID; // roll
  channels[1] = CRSF_MID; // pitch
  channels[2] = CRSF_MIN; // throttle -> low
  channels[3] = CRSF_MID; // yaw
  channels[4] = CRSF_MIN; // AUX1 / arm switch -> disarmed
}

// ----------------------------------------------------

void setup()
{
  Serial.begin(115200);

  // UART CRSF
  CRSFSerial.begin(
      CRSF_BAUDRATE,
      SERIAL_8N1,
      -1,
      CRSF_TX_PIN);

  // safety initial values: centered sticks, low throttle, disarmed
  for (int i = 0; i < 16; i++)
    channels[i] = CRSF_MID;

  applyFailsafe();

  // WIFI AP
  WiFi.softAP(ssid, password);

  // Avoid WiFi modem-sleep induced latency spikes on the control link
  WiFi.setSleep(false);

  Serial.println();
  Serial.println("WiFi AP started");
  Serial.println(WiFi.softAPIP());

  udp.begin(UDP_PORT);

  lastPacketMs = millis();
}

// ----------------------------------------------------

void loop()
{
  char buffer[64];

  // Drain ALL pending packets this loop iteration so stale packets
  // don't accumulate latency if the sender transmits faster than 50 Hz.
  int packetSize;
  while ((packetSize = udp.parsePacket()) > 0)
  {
    int len = udp.read(buffer, sizeof(buffer) - 1);

    if (len <= 0)
      continue; // nothing actually read, ignore

    buffer[len] = 0;

    // Expected format: "roll,pitch,throttle,yaw,arm"
    // roll/pitch/throttle/yaw: 172-1811 (CRSF scale)
    // arm: 0 (disarmed) or 1 (armed) -> mapped to AUX1 channel
    long roll, pitch, throttle, yaw, arm;

    int parsed = sscanf(
        buffer,
        "%ld,%ld,%ld,%ld,%ld",
        &roll,
        &pitch,
        &throttle,
        &yaw,
        &arm);

    if (parsed == 5)
    {
      channels[0] = clampChannel(roll);
      channels[1] = clampChannel(pitch);
      channels[2] = clampChannel(throttle);
      channels[3] = clampChannel(yaw);
      channels[4] = (arm != 0) ? CRSF_MAX : CRSF_MIN;

      lastPacketMs = millis();
      failsafeActive = false;
    }
    else
    {
      // Malformed packet: ignore it, do NOT touch channels[] with
      // garbage values.
      Serial.println("Bad UDP packet, ignored");
    }
  }

  // Failsafe: if we haven't heard from the controller recently,
  // force a safe state (low throttle, disarmed).
  if (!failsafeActive && (millis() - lastPacketMs > FAILSAFE_TIMEOUT_MS))
  {
    applyFailsafe();
    failsafeActive = true;
    Serial.println("Failsafe engaged: no recent UDP packets");
  }

  // Send CRSF at 50Hz
  static uint32_t last = 0;

  if (millis() - last >= 20)
  {
    last = millis();
    sendCRSF();
  }
}
