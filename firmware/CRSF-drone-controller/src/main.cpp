#include <WiFi.h>
#include <WiFiUdp.h>
#include <AlfredoCRSF.h>

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
// while true; do echo "992,992,992,992,1"; sleep 0.02; done | nc -u -w1 192.168.4.1 4210

// ----------------------------------------------------

// -------- WIFI AP --------

const char *ssid = "AMF-DRONE";
const char *password = "12345678";

WiFiUDP udp;

const uint16_t UDP_PORT = 4210;

// -------- CRSF UART --------
HardwareSerial CRSFSerial(2);
AlfredoCRSF crsf;

#define CRSF_TX_PIN 10
// If we only want to transmit, we can set RX to -1
#define CRSF_RX_PIN 9

// -------- Channel value range --------
// These are the standard CRSF 11-bit tick values, and they happen to be
// numerically identical to the SBUS ones (172 / 992 / 1811), so the UDP
// packet format and the clamping logic don't need to change.
#define CRSF_MIN CRSF_CHANNEL_VALUE_MIN
#define CRSF_MID CRSF_CHANNEL_VALUE_MID
#define CRSF_MAX CRSF_CHANNEL_VALUE_MAX

// -------- Failsafe --------
// If no valid UDP packet is received within this time, the channels are
// forced to a safe state (disarmed, throttle low).
#define FAILSAFE_TIMEOUT_MS 200

// CRSF channel packet (16 channels packed as 11-bit values).
// This struct is the exact over-the-wire layout, so it's sent as-is.
crsf_channels_t channels;

uint32_t lastPacketMs = 0;
bool failsafeActive = false;

// ----------------------------------------------------
// Send the current channel values to the flight controller as a
// CRSF_FRAMETYPE_RC_CHANNELS_PACKED frame.
// ----------------------------------------------------

void sendCRSF()
{
  // NOTE: crsf.queuePacket() silently drops the packet unless the library's
  // internal _linkIsUp flag is true, and that flag is only ever set when
  // this AlfredoCRSF instance *receives* a channels packet on its own RX
  // line. Since we're only sending (no RX wired back from the FC),
  // _linkIsUp never becomes true and queuePacket() would never transmit
  // anything - which is exactly why nothing showed up in the Betaflight
  // receiver tab. writePacket() has no such gate and always sends, so it's
  // the correct call for a one-way "channel forwarder" like this one.
  crsf.writePacket(CRSF_ADDRESS_FLIGHT_CONTROLLER, CRSF_FRAMETYPE_RC_CHANNELS_PACKED, &channels, sizeof(channels));
}

// ----------------------------------------------------
// Apply a channel value after validating its limits
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
// Set every channel to a neutral midpoint value
// ----------------------------------------------------

void setAllChannelsMid()
{
  channels.ch0 = CRSF_MID;
  channels.ch1 = CRSF_MID;
  channels.ch2 = CRSF_MID;
  channels.ch3 = CRSF_MID;
  channels.ch4 = CRSF_MID;
  channels.ch5 = CRSF_MID;
  channels.ch6 = CRSF_MID;
  channels.ch7 = CRSF_MID;
  channels.ch8 = CRSF_MID;
  channels.ch9 = CRSF_MID;
  channels.ch10 = CRSF_MID;
  channels.ch11 = CRSF_MID;
  channels.ch12 = CRSF_MID;
  channels.ch13 = CRSF_MID;
  channels.ch14 = CRSF_MID;
  channels.ch15 = CRSF_MID;
}

// ----------------------------------------------------
// Put the aircraft in a safe state
// ----------------------------------------------------

void applyFailsafe()
{
  channels.ch0 = CRSF_MID; // roll
  channels.ch1 = CRSF_MID; // pitch
  channels.ch2 = CRSF_MIN; // throttle -> low
  channels.ch3 = CRSF_MID; // yaw
  channels.ch4 = CRSF_MIN; // AUX1 / arm switch -> disarmed
}

// ----------------------------------------------------

void setup()
{
  Serial.begin(115200);

  // CRSF UART: 420000 baud, 8 data bits, no parity, 1 stop bit, not inverted.
  CRSFSerial.begin(
      CRSF_BAUDRATE,
      SERIAL_8N1,
      CRSF_RX_PIN,
      CRSF_TX_PIN);

  crsf.begin(CRSFSerial);

  // Initial safe values
  setAllChannelsMid();
  applyFailsafe();

  // WIFI AP
  WiFi.softAP(ssid, password);

  // Avoid latency spikes on the control connection
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
      channels.ch0 = clampChannel(roll);
      channels.ch1 = clampChannel(pitch);
      channels.ch2 = clampChannel(throttle);
      channels.ch3 = clampChannel(yaw);
      channels.ch4 = (arm != 0) ? CRSF_MAX : CRSF_MIN;

      lastPacketMs = millis();
      failsafeActive = false;
    }
    else
    {
      Serial.println("Bad UDP packet, ignored");
    }
  }

  // Process any data coming back from the FC (e.g. telemetry), and let the
  // library track link state. Harmless even with RX unconnected.
  crsf.update();

  // Failsafe: no recent UDP packets
  if (!failsafeActive && (millis() - lastPacketMs > FAILSAFE_TIMEOUT_MS))
  {
    applyFailsafe();
    failsafeActive = true;
    Serial.println("Failsafe engaged: no recent UDP packets");
  }

  // Send CRSF channels at 100Hz (every 10ms).
  // CRSF can go faster than SBUS (ELRS commonly runs 150-500Hz),
  // so lower this interval to have a snappier link.
  static uint32_t last = 0;

  if (millis() - last >= 10)
  {
    last = millis();
    sendCRSF();
  }
}
