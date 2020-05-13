// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rudderlabs/rudder-server/utils/misc (interfaces: PerfStatsI)

// Package mock_misc is a generated GoMock package.
package mock_misc

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockPerfStatsI is a mock of PerfStatsI interface
type MockPerfStatsI struct {
	ctrl     *gomock.Controller
	recorder *MockPerfStatsIMockRecorder
}

// MockPerfStatsIMockRecorder is the mock recorder for MockPerfStatsI
type MockPerfStatsIMockRecorder struct {
	mock *MockPerfStatsI
}

// NewMockPerfStatsI creates a new mock instance
func NewMockPerfStatsI(ctrl *gomock.Controller) *MockPerfStatsI {
	mock := &MockPerfStatsI{ctrl: ctrl}
	mock.recorder = &MockPerfStatsIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPerfStatsI) EXPECT() *MockPerfStatsIMockRecorder {
	return m.recorder
}

// End mocks base method
func (m *MockPerfStatsI) End(arg0 int) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "End", arg0)
}

// End indicates an expected call of End
func (mr *MockPerfStatsIMockRecorder) End(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "End", reflect.TypeOf((*MockPerfStatsI)(nil).End), arg0)
}

// Print mocks base method
func (m *MockPerfStatsI) Print() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Print")
}

// Print indicates an expected call of Print
func (mr *MockPerfStatsIMockRecorder) Print() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Print", reflect.TypeOf((*MockPerfStatsI)(nil).Print))
}

// Setup mocks base method
func (m *MockPerfStatsI) Setup(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Setup", arg0)
}

// Setup indicates an expected call of Setup
func (mr *MockPerfStatsIMockRecorder) Setup(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Setup", reflect.TypeOf((*MockPerfStatsI)(nil).Setup), arg0)
}

// Start mocks base method
func (m *MockPerfStatsI) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start
func (mr *MockPerfStatsIMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockPerfStatsI)(nil).Start))
}