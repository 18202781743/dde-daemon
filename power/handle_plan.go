package main

import (
	"dbus/com/deepin/daemon/display"
	"fmt"
	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/dpms"
	"time"
)

const (
	//sync with com.deepin.daemon.power.schema
	PowerPlanCustom          = 0
	PowerPlanPowerSaver      = 1
	PowerPlanBalanced        = 2
	PowerPlanHighPerformance = 3
)

func (p *Power) setBatteryIdleDelay(delay int32) {
	p.setPropBatteryIdleDelay(delay)

	if p.BatteryPlan.Get() == PowerPlanCustom && int32(p.coreSettings.GetInt("battery-idle-delay")) != delay {
		p.coreSettings.SetInt("battery-idle-delay", int(delay))
	}
	p.updateIdletimer()
	p.updatePlanInfo()
}

func (p *Power) setBatterySuspendDelay(delay int32) {
	p.setPropBatterySuspendDelay(delay)

	if p.BatteryPlan.Get() == PowerPlanCustom && int32(p.coreSettings.GetInt("battery-suspend-delay")) != delay {
		p.coreSettings.SetInt("battery-suspend-delay", int(delay))
	}
	p.updateIdletimer()
	p.updatePlanInfo()
}

func (p *Power) setBatteryPlan(plan int32) {
	switch plan {
	case PowerPlanHighPerformance:
		p.setBatteryIdleDelay(0)
		p.setBatterySuspendDelay(0)
	case PowerPlanBalanced:
		p.setBatteryIdleDelay(600)
		p.setBatterySuspendDelay(0)
	case PowerPlanPowerSaver:
		p.setBatteryIdleDelay(300)
		p.setBatterySuspendDelay(600)
	case PowerPlanCustom:
		p.setBatteryIdleDelay(int32(p.coreSettings.GetInt("battery-idle-delay")))
		p.setBatterySuspendDelay(int32(p.coreSettings.GetInt("battery-suspend-delay")))
	}
}

func (p *Power) setLinePowerIdleDelay(delay int32) {
	p.setPropLinePowerIdleDelay(delay)

	if p.LinePowerPlan.Get() == PowerPlanCustom && int32(p.coreSettings.GetInt("ac-idle-delay")) != delay {
		p.coreSettings.SetInt("ac-idle-delay", int(delay))
	}
	p.updateIdletimer()
	p.updatePlanInfo()
}

func (p *Power) setLinePowerSuspendDelay(delay int32) {
	p.setPropLinePowerSuspendDelay(delay)

	if p.LinePowerPlan.Get() == PowerPlanCustom && int32(p.coreSettings.GetInt("ac-suspend-delay")) != delay {
		p.coreSettings.SetInt("ac-suspend-delay", int(delay))
	}
	p.updateIdletimer()
	p.updatePlanInfo()
}

func (p *Power) setLinePowerPlan(plan int32) {
	switch plan {
	case PowerPlanHighPerformance:
		p.setLinePowerIdleDelay(0)
		p.setLinePowerSuspendDelay(0)
	case PowerPlanBalanced:
		p.setLinePowerIdleDelay(600)
		p.setLinePowerSuspendDelay(0)
	case PowerPlanPowerSaver:
		p.setLinePowerIdleDelay(300)
		p.setLinePowerSuspendDelay(600)
	case PowerPlanCustom:
		p.setLinePowerIdleDelay(int32(p.coreSettings.GetInt("ac-idle-delay")))
		p.setLinePowerSuspendDelay(int32(p.coreSettings.GetInt("ac-suspend-delay")))
	}
}

var suspendDelta int32 = 0

func (p *Power) updateIdletimer() {
	var min int32
	var idle, suspend int32
	if p.OnBattery {
		idle = p.BatteryIdleDelay
		suspend = p.BatterySuspendDelay
	} else {
		idle = p.LinePowerIdleDelay
		suspend = p.LinePowerSuspendDelay
	}
	if idle == 0 {
		idle = 0xfffffff
	}
	if suspend == 0 {
		suspend = 0xfffffff
	}

	if idle < suspend {
		min = idle
		suspendDelta = suspend - idle
	} else {
		min = suspend
		suspendDelta = idle - suspend
	}
	if suspendDelta > 0xfffff {
		suspendDelta = 0
	}
	if min > 0xffffff {
		min = 0
	}
	if err := p.screensaver.SetTimeout(uint32(min)/10, 0, false); err != nil {
		LOGGER.Error("Failed set ScreenSaver's timeout:", err)
	} else {
		LOGGER.Info("Set ScreenTimeout to ", uint32(min), uint32(suspendDelta))
	}
}

func (p *Power) updatePlanInfo() {
	info := fmt.Sprintf(`{
		PowerLine:{Custom:[%d,%d], PowerSaver:[300,600], Blanced:[600,0],HighPerformance:[0,0]},
		Battery:{Custom:[%d,%d], PowerSaver:[300,600], Blanced:[600,0],HighPerformance:[0,0]}
	}`, p.LinePowerIdleDelay, p.LinePowerSuspendDelay, p.BatteryIdleDelay, p.BatterySuspendDelay)
	p.setPropPlanInfo(info)
}

var dpmsOn func()
var dpmsOff func()

func (p *Power) initPlan() {
	p.screensaver.ConnectIdleOn(p.handleIdleOn)
	p.screensaver.ConnectIdleOff(p.handleIdleOff)
	p.updateIdletimer()
	con, _ := xgb.NewConn()
	dpms.Init(con)
	dpmsOn = func() { dpms.ForceLevel(con, dpms.DPMSModeOn) }
	dpmsOff = func() { dpms.ForceLevel(con, dpms.DPMSModeOff) }
}

var stopAnimation []chan bool

func doIdleAction() {
	dp, _ := display.NewDisplay("com.deepin.daemon.Display", "/com/deepin/daemon/Display")
	defer display.DestroyDisplay(dp)

	stoper := make(chan bool)
	stopAnimation = append(stopAnimation, stoper)
	for _, p := range dp.Monitors.Get() {
		go func() {
			m, _ := display.NewMonitor("com.deepin.daemon.Display", p)
			defer display.DestroyMonitor(m)

			for v := 0.8; v > 0.1; v -= 0.05 {
				<-time.After(time.Millisecond * time.Duration(float64(400)*(v)))

				select {
				case <-stoper:
					for name, _ := range m.Brightness.Get() {
						m.ResetBrightness(name)
					}
					dpmsOn()
					return

				default:
					for name, _ := range m.Brightness.Get() {
						m.ChangeBrightness(name, v)
					}
				}
			}
		}()
	}

	dpmsOff()
	if suspendDelta != 0 {
		for {
			select {
			case <-time.After(time.Second * time.Duration(suspendDelta/10)):
				doSuspend()
				return
			case <-stoper:
				return
			}
		}
	}
}

func (p *Power) handleIdleOn() {
	if p.OnBattery {
		if p.BatteryIdleDelay < p.BatterySuspendDelay || p.BatterySuspendDelay == 0 {
			doIdleAction()
		} else {
			doSuspend()
		}
	} else {
		if p.LinePowerIdleDelay < p.LinePowerSuspendDelay || p.LinePowerSuspendDelay == 0 {
			doIdleAction()
		} else {
			doSuspend()
		}
	}
}

func (*Power) handleIdleOff() {
	for _, c := range stopAnimation {
		close(c)
	}
	stopAnimation = nil

	dpmsOn()
	dp, _ := display.NewDisplay("com.deepin.daemon.Display", "/com/deepin/daemon/Display")
	defer display.DestroyDisplay(dp)
	for _, p := range dp.Monitors.Get() {
		m, _ := display.NewMonitor("com.deepin.daemon.Display", p)
		defer display.DestroyMonitor(m)
		for name, _ := range m.Brightness.Get() {
			m.ResetBrightness(name)
		}
	}
}
