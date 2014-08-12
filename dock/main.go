package dock

import (
	"flag"
	"os"
	"pkg.linuxdeepin.com/lib/dbus"
	"pkg.linuxdeepin.com/lib/log"
)

var (
	logger          = log.NewLogger("com.deepin.daemon.Dock")
	argDebug        bool
	region          *Region           = nil
	setting         *Setting          = nil
	hideModemanager *HideStateManager = nil
)

func Stop() {
	logger.EndTracing()
}
func Start() {
	logger.BeginTracing()

	initDeepin()

	// configure logger
	flag.BoolVar(&argDebug, "d", false, "debug mode")
	flag.BoolVar(&argDebug, "debug", false, "debug mode")
	flag.Parse()
	if argDebug {
		logger.SetLogLevel(log.LevelDebug)
		os.Setenv("G_MESSAGES_DEBUG", "all")
	}

	m := NewEntryProxyerManager()
	err := dbus.InstallOnSession(m)
	if err != nil {
		logger.Errorf("register dbus interface failed: %v", err)
		os.Exit(1)
	}

	m.watchEntries()

	d := NewDockedAppManager()
	err = dbus.InstallOnSession(d)
	if err != nil {
		logger.Errorf("register dbus interface failed: %v", err)
		os.Exit(1)
	}

	setting = NewSetting()
	err = dbus.InstallOnSession(setting)
	if err != nil {
		logger.Errorf("register dbus interface failed: %v", err)
		os.Exit(1)
	}

	hideModemanager = NewHideStateManager(setting.GetHideMode())
	err = dbus.InstallOnSession(hideModemanager)
	if err != nil {
		logger.Errorf("register dbus interface failed: %v", err)
		os.Exit(1)
	}
	hideModemanager.UpdateState()

	cm := NewClientManager()
	err = dbus.InstallOnSession(cm)
	if err != nil {
		logger.Errorf("register dbus interface failed: %v", err)
	}
	go cm.listenRootWindow()

	region = NewRegion()
	dbus.InstallOnSession(region)

	initialize()
}
