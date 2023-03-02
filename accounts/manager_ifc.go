// SPDX-FileCopyrightText: 2018 - 2022 UnionTech Software Technology Co., Ltd.
//
// SPDX-License-Identifier: GPL-3.0-or-later

package accounts

/*
#cgo CFLAGS: -W -Wall -g  -fstack-protector-all -fPIC
#cgo LDFLAGS: -lkeyring

#include <stdlib.h>
#include <shadow.h>
#include "keyring/common.h"
*/

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path"
	"strconv"
	"time"

	"github.com/godbus/dbus"
	"github.com/linuxdeepin/dde-daemon/accounts/checkers"
	"github.com/linuxdeepin/dde-daemon/accounts/keyring"
	"github.com/linuxdeepin/dde-daemon/accounts/users"
	"github.com/linuxdeepin/go-lib/dbusutil"
	"github.com/linuxdeepin/go-lib/gettext"
	"github.com/linuxdeepin/go-lib/procfs"
	"github.com/linuxdeepin/go-lib/users/passwd"
	dutils "github.com/linuxdeepin/go-lib/utils"
)

const (
	nilObjPath      = dbus.ObjectPath("/")
	dbusServiceName = "com.deepin.daemon.Accounts"
	dbusPath        = "/com/deepin/daemon/Accounts"
	dbusInterface   = "com.deepin.daemon.Accounts"
)

func (*Manager) GetInterfaceName() string {
	return dbusInterface
}

func createWhiteBoxUFile(name string) error {
	return keyring.CreateWhiteBoxUFile(name)
}

func getUserNameList() (list []string) {
	infos, err := users.GetHumanUserInfos()
	if err != nil {
		return nil
	}
	for _, v := range infos {
		list = append(list, v.Name)
	}
	return list
}

// 已经存在的账户，未创建WB_UFile文件，则直接创建
func (*Manager) createExistAccountWbUFile() {
	userList := getUserNameList()
	for _, name := range userList {
		dir := fmt.Sprintf("/var/lib/keyring/%s", name)
		filename := path.Join(dir, "WB_UFile")
		if !dutils.IsFileExist(dir) || !dutils.IsFileExist(filename) {
			err := keyring.CreateWhiteBoxUFile(name)
			if err != nil {
				logger.Warning("Keyring crypto so not exist.")
				return
			}
		}
	}
}

// Create new user.
//
// 如果收到 Error 信号，则创建失败。
//
// name: 用户名
//
// fullName: 全名，可以为空
//
// ty: 用户类型，0 为普通用户，1 为管理员

func (m *Manager) CreateUser(sender dbus.Sender,
	name, fullName string, accountType int32) (userPath dbus.ObjectPath, busErr *dbus.Error) {

	logger.Debug("[CreateUser] new user:", name, fullName, accountType)

	err := checkAccountType(int(accountType))
	if err != nil {
		return nilObjPath, dbusutil.ToError(err)
	}

	err = m.checkAuth(sender)
	if err != nil {
		logger.Debug("[CreateUser] access denied:", err)
		return nilObjPath, dbusutil.ToError(err)
	}

	ch := make(chan string)
	m.usersMapMu.Lock()
	m.userAddedChanMap[name] = ch
	m.usersMapMu.Unlock()
	defer func() {
		m.usersMapMu.Lock()
		delete(m.userAddedChanMap, name)
		m.usersMapMu.Unlock()
		close(ch)
	}()

	homeDir := "/home/" + name
	_, err = os.Stat(homeDir)
	homeDirExist := err == nil

	if err := users.CreateUser(name, fullName, ""); err != nil {
		logger.Warningf("DoAction: create user '%s' failed: %v\n",
			name, err)
		return nilObjPath, dbusutil.ToError(err)
	}

	groups := users.GetPresetGroups(int(accountType))
	logger.Debug("groups:", groups)
	err = users.SetGroupsForUser(groups, name)
	if err != nil {
		logger.Warningf("failed to set groups for user %s: %v", name, err)
	}

	if err = createWhiteBoxUFile(name); err != nil {
		logger.Warningf("createWhiteBoxUFile: create user '%s' failed: %v\n", name, err)
	}

	// create user success
	select {
	case userPath, ok := <-ch:
		if !ok {
			return nilObjPath, dbusutil.ToError(errors.New("invalid user path event"))
		}

		logger.Debug("receive user path", userPath)
		if userPath == "" {
			return nilObjPath, dbusutil.ToError(errors.New("failed to install user on session bus"))
		}
		if homeDirExist {
			go chownHomeDir(homeDir, name)
		}
		return dbus.ObjectPath(userPath), nil
	case <-time.After(time.Second * 60):
		err := errors.New("wait timeout exceeded")
		logger.Warning(err)
		return nilObjPath, dbusutil.ToError(err)
	}
}

// Delete a exist user.
//
// name: 用户名
//
// rmFiles: 是否删除用户数据
func (m *Manager) DeleteUser(sender dbus.Sender,
	name string, rmFiles bool) *dbus.Error {

	logger.Debug("[DeleteUser] user:", name, rmFiles)

	err := m.checkAuth(sender)
	if err != nil {
		logger.Debug("[DeleteUser] access denied:", err)
		return dbusutil.ToError(err)
	}

	user := m.getUserByName(name)
	if user == nil {
		err := fmt.Errorf("user %q not found", name)
		logger.Warning(err)
		return dbusutil.ToError(err)
	}

	if m.isDomainUser(user.Uid) {
		id, _ := strconv.Atoi(user.Uid)

		if m.udcpCache != nil && m.isUdcpUserID(user.Uid) {
			result, err := m.udcpCache.RemoveCacheFile(0, uint32(id))
			if err != nil {
				logger.Errorf("Udcp cache RemoveCacheFile failed: %v", err)
				return dbusutil.ToError(err)
			}

			if !result {
				return dbusutil.ToError(errors.New("failed to remove user cache files"))
			}
		}

		// 删除账户前先删除生物特征，避免删除账户后，用户数据找不到
		if rmFiles {
			user.clearBiometricChara()
			// 删除域用户家目录
			os.RemoveAll(user.HomeDir)
		}

		// 删除服务，更新UserList
		userPath := userDBusPathPrefix + user.Uid
		// 删除对应AD域账户配置
		if len(m.userConfig) != 0 {
			delete(m.userConfig, userPath)
			m.domainUserMapMu.Lock()
			m.saveDomainUserConfig(m.userConfig)
			m.domainUserMapMu.Unlock()
		}

		m.stopExportUser(userPath)
		m.updatePropUserList()

		// 清楚域账户本地缓存
		if rmFiles {
			user.clearData()
		}

		err = m.service.Emit(m, "UserDeleted", userPath)
		if err != nil {
			logger.Warning(err)
		}
		return dbusutil.ToError(err)
	}

	// 删除账户前先删除生物特征，避免删除账户后，用户数据找不到
	if rmFiles {
		user.clearBiometricChara()
	}
	if err := users.DeleteUser(rmFiles, name); err != nil {
		logger.Warningf("DoAction: delete user '%s' failed: %v\n",
			name, err)
		return dbusutil.ToError(err)
	}

	//delete user config and icons
	if rmFiles {
		user.clearData()
	}

	err = keyring.DeleteWhiteBoxUFile(name)
	if err != nil {
		logger.Warningf("DeleteWhiteBoxUFile '%s' failed: %v\n", name, err)
		return dbusutil.ToError(err)
	}

	return nil
}

func (m *Manager) FindUserById(uid string) (user string, busErr *dbus.Error) {
	userPath := userDBusPathPrefix + uid
	for _, v := range m.UserList {
		if v == userPath {
			return v, nil
		}
	}

	return "", dbusutil.ToError(fmt.Errorf("invalid uid: %s", uid))
}

func (m *Manager) FindUserByName(name string) (user string, busErr *dbus.Error) {
	pwd, err := passwd.GetPasswdByName(name)
	if err != nil {
		logger.Warning(err)
		pwd = &passwd.Passwd{
			Name: name,
		}
	}

	m.usersMapMu.Lock()
	defer m.usersMapMu.Unlock()

	for p, u := range m.usersMap {
		if u.UserName == pwd.Name {
			return p, nil
		}
	}

	return "", dbusutil.ToError(fmt.Errorf("invalid username: %s", pwd.Name))
}

// 随机得到一个用户头像
//
// ret0：头像路径，为空则表示获取失败
func (m *Manager) RandUserIcon() (iconFile string, busErr *dbus.Error) {
	icons := getUserStandardIcons()
	if len(icons) == 0 {
		return "", dbusutil.ToError(errors.New("Did not find any user icons"))
	}

	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(len(icons)) // #nosec G404
	return icons[idx], nil
}

func (m *Manager) isDomainUserExist(name string) bool {
	pwd, err := passwd.GetPasswdByName(name)
	if err != nil {
		return false
	}

	id := strconv.FormatUint(uint64(pwd.Uid), 10)

	return m.isUdcpUserExists(name) || users.IsLDAPDomainUserID(id)
}

// 检查用户名是否有效
//
// ret0: 是否合法
//
// ret1: 不合法原因
//
// ret2: 不合法代码
func (m *Manager) IsUsernameValid(sender dbus.Sender, name string) (valid bool,
	msg string, code int32, busErr *dbus.Error) {
	var err error
	var info *checkers.ErrorInfo

	defer func() {
		busErr = dbusutil.ToError(err)
	}()

	pid, err := m.service.GetConnPID(string(sender))
	if err != nil {
		return
	}

	p := procfs.Process(pid)
	environ, err := p.Environ()
	if err != nil {
		return
	}

	locale := environ.Get("LANG")

	// 如果新建用户使用的用户名和域用户名一致，提示用户该用户已经存在
	if m.isDomainUserExist(name) {
		info = checkers.ErrCodeExist.Error()
	} else {
		info = checkers.CheckUsernameValid(name)
		if info == nil {
			valid = true
			return
		}
	}

	msg = info.Error.Error()
	logger.Debug("locale:", locale)
	if locale != "" {
		gettext.SetLocale(gettext.LcAll, locale)
		msg = gettext.Tr(msg)
	}
	code = int32(info.Code)
	return
}

// 检测密码是否有效
//
// ret0: 是否合法
//
// ret1: 提示信息
//
// ret2: 不合法代码
func (m *Manager) IsPasswordValid(password string) (valid bool, msg string, code int32, busErr *dbus.Error) {
	releaseType := getDeepinReleaseType()
	logger.Infof("release type %q", releaseType)
	errCode := checkers.CheckPasswordValid(releaseType, password)
	return errCode.IsOk(), errCode.Prompt(), int32(errCode), nil
}

func (m *Manager) AllowGuestAccount(sender dbus.Sender, allow bool) *dbus.Error {
	err := m.checkAuth(sender)
	if err != nil {
		return dbusutil.ToError(err)
	}

	m.PropsMu.Lock()
	defer m.PropsMu.Unlock()

	if m.AllowGuest == allow {
		return nil
	}

	success := dutils.WriteKeyToKeyFile(actConfigFile,
		actConfigGroupGroup, actConfigKeyGuest, allow)
	if !success {
		return dbusutil.ToError(errors.New("enable guest user failed"))
	}

	m.AllowGuest = allow
	_ = m.emitPropChangedAllowGuest(allow)
	return nil
}

func (m *Manager) CreateGuestAccount(sender dbus.Sender) (user string, busErr *dbus.Error) {
	err := m.checkAuth(sender)
	if err != nil {
		return "", dbusutil.ToError(err)
	}

	name, err := users.CreateGuestUser()
	if err != nil {
		return "", dbusutil.ToError(err)
	}

	info, err := users.GetUserInfoByName(name)
	if err != nil {
		return "", dbusutil.ToError(err)
	}

	return userDBusPathPrefix + info.Uid, nil
}

func (m *Manager) GetGroups() (groups []string, busErr *dbus.Error) {
	groups, err := users.GetAllGroups()
	return groups, dbusutil.ToError(err)
}

func (m *Manager) GetGroupInfoByName(name string) (groupInfo string, busErr *dbus.Error) {
	info, err := users.GetGroupByName(name)
	if err != nil {
		logger.Warning(err)
		return "", dbusutil.ToError(err)
	}
	infoJson, err := json.Marshal(info)
	if err != nil {
		logger.Warning(err)
		return "", dbusutil.ToError(err)
	}
	return string(infoJson), nil
}

func (m *Manager) GetPresetGroups(accountType int32) (groups []string, busErr *dbus.Error) {
	err := checkAccountType(int(accountType))
	if err != nil {
		return nil, dbusutil.ToError(err)
	}

	groups = users.GetPresetGroups(int(accountType))
	return groups, nil
}

// 是否使能accounts服务在监听到/etc/passwd文件变化后,执行对应的属性更新和服务导出,只允许root用户操作该接口
func (m *Manager) EnablePasswdChangedHandler(sender dbus.Sender, enable bool) *dbus.Error {
	const rootUid = "0"
	uid, err := m.service.GetConnUID(string(sender))
	if err != nil {
		return dbusutil.ToError(err)
	}
	if uid != 0 {
		return dbusutil.ToError(fmt.Errorf("not allow %v call this method", sender))
	}
	m.enablePasswdChangedHandlerMu.Lock()
	defer m.enablePasswdChangedHandlerMu.Unlock()
	if m.enablePasswdChangedHandler == enable {
		return nil
	}
	m.enablePasswdChangedHandler = enable
	if enable {
		m.handleFilePasswdChanged()
		m.modifyUserConfig(userDBusPathPrefix + rootUid) // root账户的信息需要更新（deepin安装器会在后配置界面修改语言）
	}
	return nil
}

func (m *Manager) CreateGroup(sender dbus.Sender, groupName string, gid uint32, isSystem bool) *dbus.Error {
	logger.Debug("[CreateGroup] new group:", groupName)

	err := m.checkAuth(sender)
	if err != nil {
		logger.Debug("[CreateGroup] access denied:", err)
		return dbusutil.ToError(err)
	}
	args := []string{
		groupName,
		"-f",
	}
	if gid > 0 {
		args = append(args, []string{
			"-g", fmt.Sprint(gid), "-o",
		}...)
	}
	if isSystem {
		args = append(args, "-r")
	}
	cmd := exec.Command("groupadd", args...)
	logger.Debug("[CreateGroup] exec cmd is:", cmd.String())
	err = cmd.Run()
	if err != nil {
		logger.Warning(err)
		return dbusutil.ToError(err)
	}
	groupList, _ := m.GetGroups()
	m.setPropGroupList(groupList)
	return nil
}

func (m *Manager) DeleteGroup(sender dbus.Sender, groupName string, force bool) *dbus.Error {
	logger.Debug("[DeleteGroup] del group:", groupName)
	if !m.checkGroupCanChange(groupName) {
		return dbusutil.ToError(fmt.Errorf("can not delete %v", groupName))
	}
	err := m.checkAuth(sender)
	if err != nil {
		logger.Debug("[DeleteGroup] access denied:", err)
		return dbusutil.ToError(err)
	}
	args := []string{
		groupName,
	}
	if force {
		args = append(args, "-f")
	}
	cmd := exec.Command("groupdel", args...)
	logger.Debug("[DeleteGroup] exec cmd is:", cmd.String())
	err = cmd.Run()
	if err != nil {
		logger.Warning(err)
		return dbusutil.ToError(err)
	}
	groupList, _ := m.GetGroups()
	m.setPropGroupList(groupList)
	return nil
}

func (m *Manager) ModifyGroup(sender dbus.Sender, currentGroupName string, newGroupName string, newGID uint32) *dbus.Error {
	logger.Debug("[ModifyGroup] modify group :", currentGroupName)
	if newGroupName == "" && newGID <= 0 {
		return dbusutil.ToError(errors.New("invalid modify,need new name or gid"))
	}
	if !m.checkGroupCanChange(currentGroupName) {
		return dbusutil.ToError(fmt.Errorf("can not modify %v", currentGroupName))
	}
	err := m.checkAuth(sender)
	if err != nil {
		logger.Debug("[ModifyGroup] access denied:", err)
		return dbusutil.ToError(err)
	}
	args := []string{
		currentGroupName,
	}
	if newGroupName != "" {
		args = append(args, []string{
			"-n", newGroupName,
		}...)
	}
	if newGID > 0 {
		args = append(args, []string{
			"-g", fmt.Sprint(newGID), "-o",
		}...)
	}
	cmd := exec.Command("groupmod", args...)
	logger.Debug("[ModifyGroup] exec cmd is:", cmd.String())
	err = cmd.Run()
	if err != nil {
		logger.Warning(err)
		return dbusutil.ToError(err)
	}
	groupList, _ := m.GetGroups()
	m.setPropGroupList(groupList)
	return nil
}
