// Code generated by "dbusutil-gen -type Grub2,Theme,EditAuth,Fstart grub2.go theme.go edit_auth.go fstart.go"; DO NOT EDIT.

package grub2

func (v *Grub2) setPropThemeFile(value string) (changed bool) {
	if v.ThemeFile != value {
		v.ThemeFile = value
		v.emitPropChangedThemeFile(value)
		return true
	}
	return false
}

func (v *Grub2) emitPropChangedThemeFile(value string) error {
	return v.service.EmitPropertyChanged(v, "ThemeFile", value)
}

func (v *Grub2) setPropDefaultEntry(value string) (changed bool) {
	if v.DefaultEntry != value {
		v.DefaultEntry = value
		v.emitPropChangedDefaultEntry(value)
		return true
	}
	return false
}

func (v *Grub2) emitPropChangedDefaultEntry(value string) error {
	return v.service.EmitPropertyChanged(v, "DefaultEntry", value)
}

func (v *Grub2) setPropEnableTheme(value bool) (changed bool) {
	if v.EnableTheme != value {
		v.EnableTheme = value
		v.emitPropChangedEnableTheme(value)
		return true
	}
	return false
}

func (v *Grub2) emitPropChangedEnableTheme(value bool) error {
	return v.service.EmitPropertyChanged(v, "EnableTheme", value)
}

func (v *Grub2) setPropGfxmode(value string) (changed bool) {
	if v.Gfxmode != value {
		v.Gfxmode = value
		v.emitPropChangedGfxmode(value)
		return true
	}
	return false
}

func (v *Grub2) emitPropChangedGfxmode(value string) error {
	return v.service.EmitPropertyChanged(v, "Gfxmode", value)
}

func (v *Grub2) setPropTimeout(value uint32) (changed bool) {
	if v.Timeout != value {
		v.Timeout = value
		v.emitPropChangedTimeout(value)
		return true
	}
	return false
}

func (v *Grub2) emitPropChangedTimeout(value uint32) error {
	return v.service.EmitPropertyChanged(v, "Timeout", value)
}

func (v *Grub2) setPropUpdating(value bool) (changed bool) {
	if v.Updating != value {
		v.Updating = value
		v.emitPropChangedUpdating(value)
		return true
	}
	return false
}

func (v *Grub2) emitPropChangedUpdating(value bool) error {
	return v.service.EmitPropertyChanged(v, "Updating", value)
}

func (v *EditAuth) setPropEnabledUsers(value []string) {
	v.EnabledUsers = value
	v.emitPropChangedEnabledUsers(value)
}

func (v *EditAuth) emitPropChangedEnabledUsers(value []string) error {
	return v.service.EmitPropertyChanged(v, "EnabledUsers", value)
}

func (v *Fstart) setPropIsSkipGrub(value bool) (changed bool) {
	if v.IsSkipGrub != value {
		v.IsSkipGrub = value
		v.emitPropChangedIsSkipGrub(value)
		return true
	}
	return false
}

func (v *Fstart) emitPropChangedIsSkipGrub(value bool) error {
	return v.service.EmitPropertyChanged(v, "IsSkipGrub", value)
}
