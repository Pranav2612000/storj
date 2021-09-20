// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

import { PartneredSatellite } from '@/types/common';
import { APP_STATE_ACTIONS } from '@/utils/constants/actionNames';
import { AppState } from '@/utils/constants/appStateEnum';

import { APP_STATE_MUTATIONS } from '../mutationConstants';

// Object that contains all states of views
class ViewsState {
    constructor(
        public fetchState = AppState.LOADING,
        public isAddTeamMembersPopupShown = false,
        public isAccountDropdownShown = false,
        public isSelectProjectDropdownShown = false,
        public isResourcesDropdownShown = false,
        public isSettingsDropdownShown = false,
        public isEditProjectDropdownShown = false,
        public isFreeCreditsDropdownShown = false,
        public isAvailableBalanceDropdownShown = false,
        public isPeriodsDropdownShown = false,
        public isSuccessfulRegistrationShown = false,
        public isEditProfilePopupShown = false,
        public isChangePasswordPopupShown = false,
        public isPaymentSelectionShown = false,
        public isUploadCancelPopupVisible = false,
        public isSuccessfulPasswordResetShown = false,

        public setDefaultPaymentMethodID: string = "",
        public deletePaymentMethodID: string = "",
    ) {}
}

class State {
    constructor(
        public appState: ViewsState = new ViewsState(),
        public satelliteName = '',
        public partneredSatellites = new Array<PartneredSatellite>(),
        public isBetaSatellite = false,
        public couponCodeBillingUIEnabled = false,
        public couponCodeSignupUIEnabled = false,
    ){}
}

interface AppContext {
    state: State
    commit: (string, ...unknown) => void
}

export const appStateModule = {
    state: new State(),
    mutations: {
        // Mutation changing add projectMembers members popup visibility
        [APP_STATE_MUTATIONS.TOGGLE_ADD_TEAMMEMBER_POPUP](state: State): void {
            state.appState.isAddTeamMembersPopupShown = !state.appState.isAddTeamMembersPopupShown;
        },
        // Mutation changing account dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_ACCOUNT_DROPDOWN](state: State): void {
            state.appState.isAccountDropdownShown = !state.appState.isAccountDropdownShown;
        },
        // Mutation changing select project dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_SELECT_PROJECT_DROPDOWN](state: State): void {
            state.appState.isSelectProjectDropdownShown = !state.appState.isSelectProjectDropdownShown;
        },
        // Mutation changing resources dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_RESOURCES_DROPDOWN](state: State): void {
            state.appState.isResourcesDropdownShown = !state.appState.isResourcesDropdownShown;
        },
        // Mutation changing settings dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_SETTINGS_DROPDOWN](state: State): void {
            state.appState.isSettingsDropdownShown = !state.appState.isSettingsDropdownShown;
        },
        // Mutation changing edit project dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_EDIT_PROJECT_DROPDOWN](state: State): void {
            state.appState.isEditProjectDropdownShown = !state.appState.isEditProjectDropdownShown;
        },
        // Mutation changing free credits dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_FREE_CREDITS_DROPDOWN](state: State): void {
            state.appState.isFreeCreditsDropdownShown = !state.appState.isFreeCreditsDropdownShown;
        },
        // Mutation changing available balance dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_AVAILABLE_BALANCE_DROPDOWN](state: State): void {
            state.appState.isAvailableBalanceDropdownShown = !state.appState.isAvailableBalanceDropdownShown;
        },
        // Mutation changing periods dropdown visibility
        [APP_STATE_MUTATIONS.TOGGLE_PERIODS_DROPDOWN](state: State): void {
            state.appState.isPeriodsDropdownShown = !state.appState.isPeriodsDropdownShown;
        },
        // Mutation changing 'successful registration' area visibility.
        [APP_STATE_MUTATIONS.TOGGLE_SUCCESSFUL_REGISTRATION](state: State): void {
            state.appState.isSuccessfulRegistrationShown = !state.appState.isSuccessfulRegistrationShown;
        },
        // Mutation changing 'successful password reset' area visibility.
        [APP_STATE_MUTATIONS.TOGGLE_SUCCESSFUL_PASSWORD_RESET](state: State): void {
            state.appState.isSuccessfulPasswordResetShown = !state.appState.isSuccessfulPasswordResetShown;
        },
        [APP_STATE_MUTATIONS.TOGGLE_CHANGE_PASSWORD_POPUP](state: State): void {
            state.appState.isChangePasswordPopupShown = !state.appState.isChangePasswordPopupShown;
        },
        [APP_STATE_MUTATIONS.TOGGLE_EDIT_PROFILE_POPUP](state: State): void {
            state.appState.isEditProfilePopupShown = !state.appState.isEditProfilePopupShown;
        },
        [APP_STATE_MUTATIONS.TOGGLE_UPLOAD_CANCEL_POPUP](state: State): void {
            state.appState.isUploadCancelPopupVisible = !state.appState.isUploadCancelPopupVisible;
        },
        [APP_STATE_MUTATIONS.SHOW_SET_DEFAULT_PAYMENT_METHOD_POPUP](state: State, id: string): void {
            state.appState.setDefaultPaymentMethodID = id;
        },
        [APP_STATE_MUTATIONS.SHOW_DELETE_PAYMENT_METHOD_POPUP](state: State, id: string): void {
            state.appState.deletePaymentMethodID = id;
        },
        // Mutation that closes each popup/dropdown
        [APP_STATE_MUTATIONS.CLOSE_ALL](state: State): void {
            state.appState.isAccountDropdownShown = false;
            state.appState.isSelectProjectDropdownShown = false;
            state.appState.isResourcesDropdownShown = false;
            state.appState.isSettingsDropdownShown = false;
            state.appState.isEditProjectDropdownShown = false;
            state.appState.isFreeCreditsDropdownShown = false;
            state.appState.isAvailableBalanceDropdownShown = false;
            state.appState.isPeriodsDropdownShown = false;
            state.appState.isPaymentSelectionShown = false;
        },
        [APP_STATE_MUTATIONS.CHANGE_STATE](state: State, newFetchState: AppState): void {
            state.appState.fetchState = newFetchState;
        },
        // Mutation changing payment selection visibility
        [APP_STATE_MUTATIONS.TOGGLE_PAYMENT_SELECTION](state: State, value: boolean): void {
            state.appState.isPaymentSelectionShown = value;
        },
        [APP_STATE_MUTATIONS.SET_SATELLITE_NAME](state: State, satelliteName: string): void {
            state.satelliteName = satelliteName;
        },
        [APP_STATE_MUTATIONS.SET_PARTNERED_SATELLITES](state: State, partneredSatellites: PartneredSatellite[]): void {
            state.partneredSatellites = partneredSatellites;
        },
        [APP_STATE_MUTATIONS.SET_SATELLITE_STATUS](state: State, isBetaSatellite: boolean): void {
            state.isBetaSatellite = isBetaSatellite;
        },
        [APP_STATE_MUTATIONS.SET_COUPON_CODE_BILLING_UI_STATUS](state: State, couponCodeBillingUIEnabled: boolean): void {
            state.couponCodeBillingUIEnabled = couponCodeBillingUIEnabled;
        },
        [APP_STATE_MUTATIONS.SET_COUPON_CODE_SIGNUP_UI_STATUS](state: State, couponCodeSignupUIEnabled: boolean): void {
            state.couponCodeSignupUIEnabled = couponCodeSignupUIEnabled;
        },
    },
    actions: {
        // Commits mutation for changing app popups and dropdowns visibility state
        [APP_STATE_ACTIONS.TOGGLE_TEAM_MEMBERS]: function ({commit, state}: AppContext): void {
            if (!state.appState.isAddTeamMembersPopupShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_ADD_TEAMMEMBER_POPUP);
        },
        [APP_STATE_ACTIONS.TOGGLE_ACCOUNT]: function ({commit, state}: AppContext): void {
            if (!state.appState.isAccountDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_ACCOUNT_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_SELECT_PROJECT_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isSelectProjectDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_SELECT_PROJECT_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_RESOURCES_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isResourcesDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_RESOURCES_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_SETTINGS_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isSettingsDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_SETTINGS_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_EDIT_PROJECT_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isEditProjectDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_EDIT_PROJECT_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_FREE_CREDITS_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isFreeCreditsDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_FREE_CREDITS_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_AVAILABLE_BALANCE_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isAvailableBalanceDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_AVAILABLE_BALANCE_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_PERIODS_DROPDOWN]: function ({commit, state}: AppContext): void {
            if (!state.appState.isPeriodsDropdownShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_PERIODS_DROPDOWN);
        },
        [APP_STATE_ACTIONS.TOGGLE_PAYMENT_SELECTION]: function ({commit, state}: AppContext, value: boolean): void {
            if (!state.appState.isPaymentSelectionShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_PAYMENT_SELECTION, value);
        },
        [APP_STATE_ACTIONS.TOGGLE_SUCCESSFUL_REGISTRATION]: function ({commit, state}: AppContext): void {
            if (!state.appState.isSuccessfulRegistrationShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_SUCCESSFUL_REGISTRATION);
        },
        [APP_STATE_ACTIONS.TOGGLE_SUCCESSFUL_PASSWORD_RESET]: function ({commit, state}: AppContext): void {
            if (!state.appState.isSuccessfulPasswordResetShown) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.TOGGLE_SUCCESSFUL_PASSWORD_RESET);
        },
        [APP_STATE_ACTIONS.TOGGLE_CHANGE_PASSWORD_POPUP]: function ({commit}: AppContext): void {
            commit(APP_STATE_MUTATIONS.TOGGLE_CHANGE_PASSWORD_POPUP);
        },
        [APP_STATE_ACTIONS.TOGGLE_UPLOAD_CANCEL_POPUP]: function ({commit}: AppContext): void {
            commit(APP_STATE_MUTATIONS.TOGGLE_UPLOAD_CANCEL_POPUP);
        },
        [APP_STATE_ACTIONS.TOGGLE_EDIT_PROFILE_POPUP]: function ({commit}: AppContext): void {
            commit(APP_STATE_MUTATIONS.TOGGLE_EDIT_PROFILE_POPUP);
        },
        [APP_STATE_ACTIONS.SHOW_SET_DEFAULT_PAYMENT_METHOD_POPUP]: function ({commit, state}: AppContext, methodID: string): void {
            if (!state.appState.setDefaultPaymentMethodID) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.SHOW_SET_DEFAULT_PAYMENT_METHOD_POPUP, methodID);
        },
        [APP_STATE_ACTIONS.SHOW_DELETE_PAYMENT_METHOD_POPUP]: function ({commit, state}: AppContext, methodID: string): void {
            if (!state.appState.deletePaymentMethodID) {
                commit(APP_STATE_MUTATIONS.CLOSE_ALL);
            }

            commit(APP_STATE_MUTATIONS.SHOW_DELETE_PAYMENT_METHOD_POPUP, methodID);
        },
        [APP_STATE_ACTIONS.CLOSE_POPUPS]: function ({commit}: AppContext): void {
            commit(APP_STATE_MUTATIONS.CLOSE_ALL);
        },
        [APP_STATE_ACTIONS.CHANGE_STATE]: function ({commit}: AppContext, newFetchState: AppState): void {
            commit(APP_STATE_MUTATIONS.CHANGE_STATE, newFetchState);
        },
        [APP_STATE_ACTIONS.SET_SATELLITE_NAME]: function ({commit}: AppContext, satelliteName: string): void {
            commit(APP_STATE_MUTATIONS.SET_SATELLITE_NAME, satelliteName);
        },
        [APP_STATE_ACTIONS.SET_PARTNERED_SATELLITES]: function ({commit}: AppContext, partneredSatellites: PartneredSatellite[]): void {
            commit(APP_STATE_MUTATIONS.SET_PARTNERED_SATELLITES, partneredSatellites);
        },
        [APP_STATE_ACTIONS.SET_SATELLITE_STATUS]: function ({commit}: AppContext, isBetaSatellite: boolean): void {
            commit(APP_STATE_MUTATIONS.SET_SATELLITE_STATUS, isBetaSatellite);
        },
        [APP_STATE_ACTIONS.SET_COUPON_CODE_BILLING_UI_STATUS]: function ({commit}: AppContext, couponCodeBillingUIEnabled: boolean): void {
            commit(APP_STATE_MUTATIONS.SET_COUPON_CODE_BILLING_UI_STATUS, couponCodeBillingUIEnabled);
        },
        [APP_STATE_ACTIONS.SET_COUPON_CODE_SIGNUP_UI_STATUS]: function ({commit}: AppContext, couponCodeSignupUIEnabled: boolean): void {
            commit(APP_STATE_MUTATIONS.SET_COUPON_CODE_SIGNUP_UI_STATUS, couponCodeSignupUIEnabled);
        },
    },
};
