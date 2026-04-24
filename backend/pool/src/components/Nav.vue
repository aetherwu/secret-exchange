<template>
    <div class="nav">
        <router-link to="/">Pool</router-link> |
        <router-link to="/question/list">Question</router-link> |
        <router-link to="/answer/list">Answer</router-link> |
        <router-link to="/activity/list">Activity</router-link> |
        <router-link to="/user/list">User</router-link> |
        <router-link to="/group/list">Group</router-link> |
        <router-link to="/group/topics">Topic</router-link>
    </div>
</template>

<script>
export default {
  data() {
    return {}
  }
}
</script>

<style scoped>
.nav {
  width: 100%;
  height: 64px;
}
</style>
