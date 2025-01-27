// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

import Router from 'vue-router';
import Vuex from 'vuex';

import NavigationArea from '@/components/navigation/NavigationArea.vue';
import OnboardingTourArea from '@/components/onboardingTour/OnboardingTourArea.vue';
import ProjectDashboard from '@/components/project/ProjectDashboard.vue';

import { RouteConfig } from '@/router';
import { makeProjectsModule, PROJECTS_MUTATIONS } from '@/store/modules/projects';
import { NavigationLink } from '@/types/navigation';
import { Project } from '@/types/projects';
import { createLocalVue, shallowMount } from '@vue/test-utils';

import { ProjectsApiMock } from '../mock/api/projects';

const api = new ProjectsApiMock();
api.setMockProjects([new Project('1')]);
const projectsModule = makeProjectsModule(api);
const localVue = createLocalVue();

localVue.use(Vuex);
localVue.use(Router);

const store = new Vuex.Store({ modules: { projectsModule } });

const expectedLinks: NavigationLink[] = [
    RouteConfig.ProjectDashboard,
    RouteConfig.Buckets,
    RouteConfig.AccessGrants,
    RouteConfig.Users,
];

describe('NavigationArea', () => {
    it('snapshot not changed during onboarding tour', (): void => {
        const router = new Router({
            mode: 'history',
            routes: [{
                path: '/onboarding-tour',
                name: RouteConfig.OnboardingTour.name,
                component: OnboardingTourArea,
            }],
        });

        router.push('/onboarding-tour');

        const wrapper = shallowMount(NavigationArea, {
            store,
            localVue,
            router,
        });

        const navigationElements = wrapper.findAll('.navigation-area__item-container');

        expect(navigationElements.length).toBe(0);
        expect(wrapper).toMatchSnapshot();
    });

    const router = new Router({
        mode: 'history',
        routes: [{
            path: '/',
            name: RouteConfig.ProjectDashboard.name,
            component: ProjectDashboard,
        }],
    });

    // TODO: enable when objects page will be finished
    it.skip('snapshot not changed with project', async () => {
        const projects = await store.dispatch('fetchProjects');
        store.commit(PROJECTS_MUTATIONS.SELECT_PROJECT, projects[0].id);

        router.push('/');

        const wrapper = shallowMount(NavigationArea, {
            store,
            localVue,
            router,
        });

        const navigationElements = wrapper.findAll('.navigation-area__item-container');

        expect(navigationElements.length).toBe(4);
        expect(wrapper).toMatchSnapshot();
    });

    // TODO: enable when objects page will be finished
    it.skip('navigation links are correct', () => {
        const wrapper = shallowMount(NavigationArea, {
            store,
            localVue,
            router,
        });

        const navigationLinks = wrapper.vm.navigation;

        expect(navigationLinks.length).toBe(expectedLinks.length);

        expectedLinks.forEach((_link: NavigationLink, i: number) => {
            expect(navigationLinks[i].name).toBe(expectedLinks[i].name);
            expect(navigationLinks[i].path).toBe(expectedLinks[i].path);
        });
    });
});
