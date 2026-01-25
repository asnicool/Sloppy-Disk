import { createApp } from 'vue'
import router from './router'
import App from './App.vue'

// Global styles
import './styles/main.css'

const app = createApp(App)

app.use(router)

// Global error handler
app.config.errorHandler = (err, vm, info) => {
  console.error('Global error:', err, info)
}

app.mount('#app')