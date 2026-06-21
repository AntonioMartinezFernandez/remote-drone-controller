package main

import (
	"fmt"
	"net"
	"time"
)

const espAddr = "192.168.4.1:4210"

// CRSF channel ticks - same range used by the ESP32 sketch.
const (
	crsfMin = 172
	crsfMid = 992
	crsfMax = 1811
)

const (
	sendInterval = 20 * time.Millisecond  // stay well under the 200ms failsafe timeout
	armHoldTime  = 800 * time.Millisecond // give the FC time to register the arm switch at low throttle
	rampDuration = 5 * time.Second        // smooth ramp time from min throttle to takeoff throttle
)

const takeoffThrottle = 900 // gentle takeoff value -- tune to your drone/props

func send(conn net.Conn, roll, pitch, throttle, yaw, arm int) {
	msg := fmt.Sprintf("%d,%d,%d,%d,%d", roll, pitch, throttle, yaw, arm)
	if _, err := conn.Write([]byte(msg)); err != nil {
		fmt.Println("send error:", err)
	}
}

func main() {
	conn, err := net.Dial("udp", espAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	ticker := time.NewTicker(sendInterval)
	defer ticker.Stop()

	// 1. Arm at minimum throttle, hold briefly so the FC registers it.
	// Betaflight requires throttle to be low at the moment of arming.
	fmt.Println("Arming (throttle low)...")
	for deadline := time.Now().Add(armHoldTime); time.Now().Before(deadline); {
		<-ticker.C
		send(conn, crsfMid, crsfMid, crsfMin, crsfMid, 1)
	}

	// 2. Smoothly ramp throttle up to the takeoff value
	fmt.Println("Ramping throttle for takeoff...")
	steps := int(rampDuration / sendInterval)
	for i := 0; i <= steps; i++ {
		<-ticker.C
		throttle := crsfMin + (takeoffThrottle-crsfMin)*i/steps
		send(conn, crsfMid, crsfMid, throttle, crsfMid, 1)
	}

	// 3. Hold takeoff throttle, keep streaming so the failsafe doesn't trip.
	// Ctrl+C stops this program; once packets stop arriving, the ESP32's
	// own 200ms failsafe takes over and disarms/zeros throttle automatically.
	fmt.Println("Holding throttle. Press Ctrl+C to stop.")
	for {
		<-ticker.C
		send(conn, crsfMid, crsfMid, takeoffThrottle, crsfMid, 1)
	}
}
