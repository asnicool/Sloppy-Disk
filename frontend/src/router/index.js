import { createRouter, createWebHistory } from 'vue-router'

// Import views
import AlbumsView from '@/views/AlbumsView.vue'
import ArtistsView from '@/views/ArtistsView.vue'
import AlbumDetailView from '@/views/AlbumDetailView.vue'
import SearchView from '@/views/SearchView.vue'
import LibraryView from '@/views/LibraryView.vue'
import DatesView from '@/views/DatesView.vue'
import GenresView from '@/views/GenresView.vue'
import NowPlayingView from '@/views/NowPlayingView.vue'
import QueueView from '@/views/QueueView.vue'

const routes = [
  {
    path: '/',
    name: 'albums-root',
    redirect: '/albums'
  },
  {
    path: '/albums',
    name: 'albums',
    component: AlbumsView
  },
  {
    path: '/albums/:artist/:album',
    name: 'album-detail',
    component: AlbumDetailView,
    props: true
  },
  {
    path: '/artists',
    name: 'artists',
    component: ArtistsView
  },
  {
    path: '/search',
    name: 'search',
    component: SearchView
  },
  {
    path: '/library',
    name: 'library',
    component: LibraryView
  },
  {
    path: '/dates',
    name: 'dates',
    component: DatesView
  },
  {
    path: '/genres',
    name: 'genres',
    component: GenresView
  },
  {
    path: '/nowplaying',
    name: 'nowplaying',
    component: NowPlayingView
  },
  {
    path: '/queue',
    name: 'queue',
    component: QueueView
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router