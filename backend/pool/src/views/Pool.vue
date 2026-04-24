<template>
  <div class="pool">
    <Nav></Nav>
    <div><router-link to="/question/list">问题：{{questions}} 个</router-link></div>
    <div><router-link to="/answer/list">答案：{{answers}} 个</router-link></div>
    <div><router-link to="/activity/list">交换：{{exchanges}} 次</router-link></div>
    <div><router-link to="/user/list">用户：{{users}}个</router-link></div>
    <div><router-link to="/group/list">群组：{{groups}}个</router-link></div>
    <div><router-link to="/group/topics">群主题：{{groupTopics}}个</router-link></div>
  </div>
</template>


<script>
import config from '../config'
import Nav from '@/components/Nav'

export default {
  components: {
    Nav
  },
  data() {
    return {
      answers: 0,
      exchanges: 0,
      questions: 0,
      users: 0,
      groups: 0,
      groupTopics: 0,
    }
  },
  methods: {
    reloadData() {
      this.$http
        .post(config.service.baseURL + '/dashboard/count', {})
        .then(resp => {
          this.answers = resp.data.answers
          this.exchanges = resp.data.exchanges
          this.questions = resp.data.questions
          this.users = resp.data.users
          this.groups = resp.data.groups
          this.groupTopics = resp.data.groupTopics
        })
    }
  },
  created() {
    this.reloadData()
  }
}
</script>
