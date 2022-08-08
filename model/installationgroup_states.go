// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

const (
	// InstallationGroupStable is an installation group in a stable state and undergoing no changes.
	InstallationGroupStable = "stable"
	// InstallationGroupReleasePending is an installation group pending release.
	InstallationGroupReleasePending = "release-pending"
	// InstallationGroupReleaseRequested is an installation group with a release requested.
	InstallationGroupReleaseRequested = "release-requested"
	// InstallationGroupReleaseSoakingRequested is an installation group with a release in soaking.
	InstallationGroupReleaseSoakingRequested = "release-soaking-requested"
	// InstallationGroupReleaseFailed is an installation group with a release in failed state.
	InstallationGroupReleaseFailed = "release-failed"
	// InstallationGroupReleaseSoakingFailed is an installation group with a soaking in failed state.
	InstallationGroupReleaseSoakingFailed = "soaking-failed"
)

// AllInstallationGroupStates is a list of all states an installation group can be in.
// Warning:
// When creating a new installation group state, it must be added to this list.
var AllInstallationGroupStates = []string{
	InstallationGroupStable,
	InstallationGroupReleasePending,
	InstallationGroupReleaseRequested,
	InstallationGroupReleaseSoakingRequested,
	InstallationGroupReleaseFailed,
	InstallationGroupReleaseSoakingFailed,
}

// AllInstallationGroupStatesPendingWork is a list of all installation group states that the supervisor
// will attempt to transition towards stable on the next "tick".
// Warning:
// When creating a new installation group state, it must be added to this list if the elrond
// supervisor should perform some action on its next work cycle.
var AllInstallationGroupStatesPendingWork = []string{
	InstallationGroupReleasePending,
	InstallationGroupReleaseRequested,
	InstallationGroupReleaseSoakingRequested,
}

// AllInstallationGroupStatesReleaseInProgress is a list of all installation group states that are part of a release in progress.
var AllInstallationGroupStatesReleaseInProgress = []string{
	InstallationGroupReleaseRequested,
	InstallationGroupReleaseSoakingRequested,
}

// AllInstallationGroupRequestStates is a list of all states that an installation group can be put in
// via the API.
// Warning:
// When creating a new installation group state, it must be added to this list if an API
// endpoint should put the installation group in this state.
var AllInstallationGroupRequestStates = []string{
	InstallationGroupReleasePending,
	InstallationGroupReleaseRequested,
	InstallationGroupReleaseSoakingRequested,
}

// ValidInstallationGroupTransitionState returns whether an installation group can be transitioned into the
// new state or not based on its current state.
func (i *InstallationGroup) ValidInstallationGroupTransitionState(newState string) bool {
	switch newState {
	case InstallationGroupReleasePending:
		return validTransitionToInstallationGroupStateReleasePending(i.State)
	case InstallationGroupReleaseRequested:
		return validTransitionToInstallationGroupStateReleaseInProgress(i.State)
	case InstallationGroupReleaseSoakingRequested:
		return validTransitionToInstallationGroupStateReleaseSoaking(i.State)
	}

	return false
}

func validTransitionToInstallationGroupStateReleasePending(currentState string) bool {
	switch currentState {
	case InstallationGroupStable,
		InstallationGroupReleasePending,
		InstallationGroupReleaseRequested,
		InstallationGroupReleaseFailed,
		InstallationGroupReleaseSoakingFailed:
		return true
	}

	return false
}

func validTransitionToInstallationGroupStateReleaseInProgress(currentState string) bool {
	switch currentState {
	case InstallationGroupReleaseRequested,
		InstallationGroupReleaseFailed,
		InstallationGroupReleaseSoakingFailed:
		return true
	}

	return false
}

func validTransitionToInstallationGroupStateReleaseSoaking(currentState string) bool {
	switch currentState {
	case InstallationGroupReleaseRequested:
		return true
	}

	return false
}

// InstallationGroupStateReport is a report of all installation group requests states.
type InstallationGroupStateReport []StateReportEntry

// GetInstallationGroupRequestStateReport returns a InstallationGroupStateReport based on the current
// model of installation group states.
func GetInstallationGroupRequestStateReport() InstallationGroupStateReport {
	report := InstallationGroupStateReport{}

	for _, requestState := range AllInstallationGroupRequestStates {
		entry := StateReportEntry{
			RequestedState: requestState,
		}

		for _, newState := range AllInstallationGroupRequestStates {
			c := InstallationGroup{State: newState}
			if c.ValidInstallationGroupTransitionState(requestState) {
				entry.ValidStates = append(entry.ValidStates, newState)
			} else {
				entry.InvalidStates = append(entry.InvalidStates, newState)
			}
		}

		report = append(report, entry)
	}

	return report
}
