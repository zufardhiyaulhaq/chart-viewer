// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	model "chart-viewer/model"

	mock "github.com/stretchr/testify/mock"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// GetCharts provides a mock function with given fields: repoName
func (_m *Service) GetCharts(repoName string) []model.Chart {
	ret := _m.Called(repoName)

	var r0 []model.Chart
	if rf, ok := ret.Get(0).(func(string) []model.Chart); ok {
		r0 = rf(repoName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Chart)
		}
	}

	return r0
}

// GetRepos provides a mock function with given fields:
func (_m *Service) GetRepos() []model.Repo {
	ret := _m.Called()

	var r0 []model.Repo
	if rf, ok := ret.Get(0).(func() []model.Repo); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Repo)
		}
	}

	return r0
}

// GetStringifiedManifests provides a mock function with given fields: repoName, chartName, chartVersion, hash
func (_m *Service) GetStringifiedManifests(repoName string, chartName string, chartVersion string, hash string) string {
	ret := _m.Called(repoName, chartName, chartVersion, hash)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string, string, string) string); ok {
		r0 = rf(repoName, chartName, chartVersion, hash)
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetTemplates provides a mock function with given fields: repoName, chartName, chartVersion
func (_m *Service) GetTemplates(repoName string, chartName string, chartVersion string) []model.Template {
	ret := _m.Called(repoName, chartName, chartVersion)

	var r0 []model.Template
	if rf, ok := ret.Get(0).(func(string, string, string) []model.Template); ok {
		r0 = rf(repoName, chartName, chartVersion)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]model.Template)
		}
	}

	return r0
}

// GetValues provides a mock function with given fields: repoName, chartName, chartVersion
func (_m *Service) GetValues(repoName string, chartName string, chartVersion string) map[string]interface{} {
	ret := _m.Called(repoName, chartName, chartVersion)

	var r0 map[string]interface{}
	if rf, ok := ret.Get(0).(func(string, string, string) map[string]interface{}); ok {
		r0 = rf(repoName, chartName, chartVersion)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]interface{})
		}
	}

	return r0
}

// RenderManifest provides a mock function with given fields: repoName, chartName, chartVersion, values
func (_m *Service) RenderManifest(repoName string, chartName string, chartVersion string, values []string) (error, model.ManifestResponse) {
	ret := _m.Called(repoName, chartName, chartVersion, values)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, string, string, []string) error); ok {
		r0 = rf(repoName, chartName, chartVersion, values)
	} else {
		r0 = ret.Error(0)
	}

	var r1 model.ManifestResponse
	if rf, ok := ret.Get(1).(func(string, string, string, []string) model.ManifestResponse); ok {
		r1 = rf(repoName, chartName, chartVersion, values)
	} else {
		r1 = ret.Get(1).(model.ManifestResponse)
	}

	return r0, r1
}