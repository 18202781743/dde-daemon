/*
 * Copyright (C) 2020 ~ 2021 Deepin Technology Co., Ltd.
 *
 * Author:     weizhixiang <1138871845@qq.com>
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package gesture

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	configPath = "testdata/gesture"
)

func findGestureInfo(name, direction string, fingers int32, infos gestureInfos) bool {
	for _, info := range infos {
		if info.Name == name && info.Direction == direction && info.Fingers == fingers {
			return true
		}
	}
	return false
}

func Test_newGestureInfosFromFile(t *testing.T) {
	infos, err := newGestureInfosFromFile(configPath)
	assert.Nil(t, err)

	assert.True(t, findGestureInfo("swipe", "up", 3, infos))
	assert.True(t, findGestureInfo("swipe", "down", 3, infos))
	assert.True(t, findGestureInfo("swipe", "left", 3, infos))
	assert.True(t, findGestureInfo("swipe", "right", 3, infos))
	assert.True(t, findGestureInfo("swipe", "up", 4, infos))
	assert.True(t, findGestureInfo("swipe", "down", 4, infos))
	assert.True(t, findGestureInfo("swipe", "left", 4, infos))
	assert.True(t, findGestureInfo("swipe", "right", 4, infos))
	assert.True(t, findGestureInfo("swipe", "up", 5, infos))
	assert.True(t, findGestureInfo("swipe", "down", 5, infos))
	assert.True(t, findGestureInfo("swipe", "left", 5, infos))
	assert.True(t, findGestureInfo("swipe", "down", 5, infos))
}

func Test_Get(t *testing.T) {
	infos, err := newGestureInfosFromFile(configPath)
	assert.Nil(t, err)

	// for touch long press
	infos = append(infos, &gestureInfo{
		Name:      "touch right button",
		Direction: "down",
		Fingers:   0,
		Action: ActionInfo{
			Type:   ActionTypeCommandline,
			Action: "xdotool mousedown 3",
		},
	})
	infos = append(infos, &gestureInfo{
		Name:      "touch right button",
		Direction: "up",
		Fingers:   0,
		Action: ActionInfo{
			Type:   ActionTypeCommandline,
			Action: "xdotool mouseup 3",
		},
	})
	assert.Nil(t, err)
	assert.NotNil(t, infos.Get("touch right button", "up", 0))
	assert.NotNil(t, infos.Get("touch right button", "up", 0))
	assert.NotNil(t, infos.Get("swipe", "up", 3))
	assert.NotNil(t, infos.Get("swipe", "down", 3))
	assert.NotNil(t, infos.Get("swipe", "left", 3))
	assert.NotNil(t, infos.Get("swipe", "right", 3))
	assert.NotNil(t, infos.Get("swipe", "up", 4))
	assert.NotNil(t, infos.Get("swipe", "down", 4))
	assert.NotNil(t, infos.Get("swipe", "left", 4))
	assert.NotNil(t, infos.Get("swipe", "right", 4))
	assert.NotNil(t, infos.Get("swipe", "up", 5))
	assert.NotNil(t, infos.Get("swipe", "down", 5))
	assert.NotNil(t, infos.Get("swipe", "left", 5))
	assert.NotNil(t, infos.Get("swipe", "right", 5))
}

func Test_Set(t *testing.T) {
	infos, err := newGestureInfosFromFile(configPath)
	assert.Nil(t, err)

	action1 := ActionInfo{
		Type:   "shortcut",
		Action: "ctrl+minus",
	}
	action2 := ActionInfo{
		Type:   "shortcut",
		Action: "ctrl+find",
	}
	assert.NotNil(t, infos.Set("pinch", "in", 2, action1))
	assert.NotNil(t, infos.Set("pinch", "out", 2, action2))
	assert.Nil(t, infos.Set("swipe", "up", 3, action1))
	assert.Nil(t, infos.Set("swipe", "down", 3, action2))
}
