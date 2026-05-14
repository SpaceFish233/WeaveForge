import { createRouter, createWebHashHistory } from 'vue-router'
import EditorView from '../views/EditorView.vue'
import SettingsView from '../views/SettingsView.vue'
import ForeshadowView from '../views/ForeshadowView.vue'
import PlotEngineView from '../views/PlotEngineView.vue'
import APISettingsView from '../views/APISettingsView.vue'
import CharacterView from '../views/CharacterView.vue'
import TimelineView from '../views/TimelineView.vue'
import OutlineView from '../views/OutlineView.vue'
import StatsView from '../views/StatsView.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'editor', component: EditorView },
    { path: '/settings', name: 'settings', component: SettingsView },
    { path: '/api', name: 'api', component: APISettingsView },
    { path: '/foreshadow', name: 'foreshadow', component: ForeshadowView },
    { path: '/plot', name: 'plot', component: PlotEngineView },
    { path: '/characters', name: 'characters', component: CharacterView },
    { path: '/timeline', name: 'timeline', component: TimelineView },
    { path: '/outline', name: 'outline', component: OutlineView },
    { path: '/stats', name: 'stats', component: StatsView },
  ],
})

export default router
