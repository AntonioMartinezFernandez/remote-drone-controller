package drone

import (
	"fmt"
	"sync"
)

type DroneState struct {
	mu       sync.Mutex
	roll     int
	pitch    int
	yaw      int
	throttle int
	arm      bool

	csrfMin int
	csrfMid int
	csrfMax int
}

func NewDroneState(
	csrfMin int,
	csrfMid int,
	csrfMax int,
) *DroneState {
	return &DroneState{
		roll:     50,
		pitch:    50,
		yaw:      50,
		throttle: 0,
		arm:      false,
		csrfMin:  csrfMin,
		csrfMid:  csrfMid,
		csrfMax:  csrfMax,
	}
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (d *DroneState) SetArm(arm bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.arm = arm
}

func (d *DroneState) SetFromPercentages(roll, pitch, yaw, throttle float64) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.roll = clamp(int(roll), 0, 100)
	d.pitch = clamp(int(pitch), 0, 100)
	d.yaw = clamp(int(yaw), 0, 100)
	d.throttle = clamp(int(throttle), 0, 100)
}

func (d *DroneState) AdjustRoll(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.roll = clamp(d.roll+delta, 0, 100)
}

func (d *DroneState) AdjustPitch(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.pitch = clamp(d.pitch+delta, 0, 100)
}

// increaseThrottle raises the throttle and arms the drone if it isn't
// already armed -- see the package-level NOTE for why.
func (d *DroneState) IncreaseThrottle(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.arm = true
	d.throttle = clamp(d.throttle+delta, 0, 100)
}

func (d *DroneState) DecreaseThrottle(delta int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttle = clamp(d.throttle-delta, 0, 100)
}

// emergencyStop immediately disarms the drone and zeroes the throttle.
func (d *DroneState) EmergencyStop() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.throttle = 0
	d.arm = false
}

// snapshot returns a consistent copy of all fields under a single lock,
// so callers (like the redraw loop) never read a half-updated state.
func (d *DroneState) Snapshot() (roll, pitch, yaw, throttle int, arm bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.roll, d.pitch, d.yaw, d.throttle, d.arm
}

// percentToCRSF linearly maps a 0-100 percentage to the CRSF tick range
// used by the firmware, where 50% maps to the centered tick value.
func (d *DroneState) percentToCRSF(percent int) int {
	percent = clamp(percent, 0, 100)
	return d.csrfMin + ((d.csrfMax-d.csrfMin)*percent+50)/100
}

func (d *DroneState) SerializePacket() string {
	d.mu.Lock()
	defer d.mu.Unlock()

	armBit := 0
	if d.arm {
		armBit = 1
	}

	return fmt.Sprintf("%d,%d,%d,%d,%d",
		d.percentToCRSF(d.roll),
		d.percentToCRSF(d.pitch),
		d.percentToCRSF(d.throttle),
		d.percentToCRSF(d.yaw),
		armBit,
	)
}
