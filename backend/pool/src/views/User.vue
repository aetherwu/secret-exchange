<template>
  <div>
    <Nav></Nav>
    <div v-if="user">
      <img class="avatar" :src="user.avatar"></img>
      <div>昵称：{{user.nickname}}</div>
      <div>答过的问题数：{{user.answerCount}}</div>
      <div>交换到的答案数{{user.exchangeCount}}</div>
      <div>最近登录时间：{{user.updatedAt}}</div>

      <el-table :data="user.qas" @cell-click="toQuestion" @sort-change="sortChange" :default-sort="{prop: 'answer.createdAt', order: 'descending'}" stripe>
        <el-table-column prop="question.content" label="问题"></el-table-column>
        <el-table-column prop="answer.content" label="答案"></el-table-column>
        <el-table-column prop="answer.createdAt" width="150" label="回答时间" sortable="custom"></el-table-column>
      </el-table>

      <el-pagination layout="prev, pager, next" :total="total" :page-size="20" @current-change="pageChange"></el-pagination>
    </div>
  </div>
</template>

<script>
import config from '../config'
import util from '../utils/util'
import Nav from '@/components/Nav'

export default {
  components: {
    Nav
  },
  data() {
    return {
      openid: '',
      user: null,
      page: 1,
      total: 0,
      request: {}
    }
  },
  methods: {
    relpadData() {
      this.request.openID = this.openid
      this.$http
        .post(config.service.baseURL + '/dashboard/users/detail', this.request)
        .then(resp => {
          resp.data.updatedAt = util.formatTimeStirng(resp.data.updatedAt)
          for (var qa of resp.data.qas) {
            qa.answer.createdAt = util.formatTimeStirng(qa.answer.createdAt)
          }
          this.user = resp.data
          this.total = resp.data.total
        })
    },
    toQuestion(row, column) {
      if (column.label === '问题') {
        this.$router.push('/question/' + row.question.id)
      }
    },
    sortChange(order) {
      var d = 0
      if (order.order === 'ascending') {
        d = 1
      } else if (order.order === 'descending') {
        d = -1
      }
      if (order.prop === 'answer.createdAt') {
        order.prop = 'createdAt'
      }
      this.request = { page: this.page, orderBy: order.prop, orderDirection: d }
      this.relpadData()
    },
    pageChange(page) {
      this.page = page
      this.request.page = page
      this.relpadData()
    }
  },
  created() {
    this.openid = this.$route.params.id
    this.relpadData()
  }
}
</script>

<style scoped>
.avatar {
  width: 50px;
  height: 50px;
}
</style>
