import { createRouter, createWebHistory } from 'vue-router'
import OrderDetail from './views/OrderDetail.vue'
import OrderList from './views/OrderList.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: OrderList },
    { path: '/orders/:id', component: OrderDetail },
  ],
})
