<template>
    <div>
        <Nav></Nav>
        <el-table :data="answers" @cell-click="pushTo" @sort-change="sortChange" :default-sort="{prop: 'answer.createdAt', order: 'descending'}" stripe>
            <el-table-column prop="answer.user.nickname" label="用户" width="150"></el-table-column>
            <el-table-column prop="question.content" label="问题"></el-table-column>
            <el-table-column prop="answer.content" label="答案"></el-table-column>
            <el-table-column prop="answer.allExchangeCount" label="被交换次数" width="150" sortable="custom"></el-table-column>
            <el-table-column prop="answer.createdAt" label="回答时间" sortable="custom"></el-table-column>
        </el-table>
        <el-pagination layout="prev, pager, next" :total="total" :page-size="20" @current-change="pageChange"></el-pagination>
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
      answers: [],
      page: 1,
      total: 0,
      request: {}
    }
  },
  methods: {
    relpadData() {
      this.$http
        .post(config.service.baseURL + '/dashboard/answers', this.request)
        .then(resp => {
          for (var a of resp.data.answers) {
            a.answer.createdAt = util.formatTimeStirng(a.answer.createdAt)
          }
          this.answers = resp.data.answers
          this.total = resp.data.total
        })
    },
    pushTo(row, column) {
      if (column.label === '用户') {
        this.$router.push('/user/' + row.answer.user.openID)
      } else if (column.label === '问题') {
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
      var by = ''
      if (order.prop === 'answer.allExchangeCount') {
        by = 'allExchangeCount'
      }
      this.request = { page: this.page, orderBy: by, orderDirection: d }
      this.relpadData()
    },
    pageChange(page) {
      this.page = page
      this.request.page = page
      this.relpadData()
    }
  }
}
</script>

<style scoped>
.avatar {
  width: 50px;
  height: 50px;
}
</style>
