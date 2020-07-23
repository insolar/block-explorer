// Copyright 2020 Insolar Network Ltd.
// All rights reserved.
// This material is licensed under the Insolar License version 1.0,
// available at https://github.com/insolar/block-explorer/blob/master/LICENSE.md.

package common

// StopChannel is the global channel where you have to send signal to stop application
var StopChannel = make(chan struct{})
