// Code generated by "dbusutil-gen -type Manager manager.go"; DO NOT EDIT.

package timedate1

func (v *Manager) setPropNTPServer(value string) (changed bool) {
	if v.NTPServer != value {
		v.NTPServer = value
		v.emitPropChangedNTPServer(value)
		return true
	}
	return false
}

func (v *Manager) emitPropChangedNTPServer(value string) error {
	return v.service.EmitPropertyChanged(v, "NTPServer", value)
}