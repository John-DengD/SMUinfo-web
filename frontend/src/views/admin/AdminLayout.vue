<template>
  <div class="admin-layout">
    <aside class="aside">
      <div class="title">管理后台</div>
      <el-menu :default-active="route.path" router>
        <el-menu-item index="/admin/users">用户管理</el-menu-item>
        <el-menu-item index="/admin/products">商品管理</el-menu-item>
        <el-menu-item index="/admin/categories">分类管理</el-menu-item>
        <el-menu-item index="/admin/reports">举报管理</el-menu-item>
        <el-menu-item index="/admin/feedback">意见箱</el-menu-item>
        <el-menu-item index="/admin/announcements">公告管理</el-menu-item>
      </el-menu>
    </aside>
    <section class="main">
      <router-view />
    </section>
  </div>
</template>

<script setup>
import { useRoute } from 'vue-router'
const route = useRoute()
</script>

<style scoped>
.admin-layout {
  display: grid;
  grid-template-columns: 220px 1fr;
  min-height: calc(100vh - 64px);
  background: #f5f7fa;
}
.aside {
  background: #fff;
  border-right: 1px solid var(--border);
}
.title {
  padding: 16px;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid var(--border);
}
.main { padding: 20px; }
</style>
