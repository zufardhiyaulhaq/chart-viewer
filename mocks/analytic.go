// Code generated by mockery v2.32.0. DO NOT EDIT.

package mocks

import (
	model "chart-viewer/pkg/model"

	mock "github.com/stretchr/testify/mock"
)

// Analytic is an autogenerated mock type for the Analytic type
type Analytic struct {
	mock.Mock
}

// Analyze provides a mock function with given fields: templates, kubeAPIVersions
func (_m *Analytic) Analyze(templates []model.Template, kubeAPIVersions model.KubernetesAPIVersion) ([]model.AnalyticsResult, error) {
	ret := _m.Called(templates, kubeAPIVersions)

	var r0 []model.AnalyticsResult
	var r1 error
	if rf, ok := ret.Get(0).(func([]model.Template, model.KubernetesAPIVersion) ([]model.AnalyticsResult, error)); ok {
		return rf(templates, kubeAPIVersions)
	}
	if rf, ok := ret.Get(0).(func([]model.Template, model.KubernetesAPIVersion) []model.AnalyticsResult); ok {
		r0 = rf(templates, kubeAPIVersions)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.AnalyticsResult)
		}
	}

	if rf, ok := ret.Get(1).(func([]model.Template, model.KubernetesAPIVersion) error); ok {
		r1 = rf(templates, kubeAPIVersions)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewAnalytic creates a new instance of Analytic. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewAnalytic(t interface {
	mock.TestingT
	Cleanup(func())
}) *Analytic {
	mock := &Analytic{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}