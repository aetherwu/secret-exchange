<template>
  <div>
    <Nav></Nav>
    <el-table :data="users" @row-click="toDetail"  @sort-change="sortChange" :default-sort="{prop: 'updatedAt', order: 'descending'}" stripe>
      <el-table-column label="头像" width="80">
        <template slot-scope="scope">
          <img class="avatar" :src="scope.row.avatar" />
        </template>
      </el-table-column>
      <el-table-column prop="nickname" label="昵称" width="150"></el-table-column>
      <el-table-column prop="answerCount" label="答过的问题数" width="150" sortable="custom"></el-table-column>
      <el-table-column prop="exchangeCount" label="交换到的答案数" width="150" sortable="custom"></el-table-column>
      <el-table-column prop="updatedAt" label="最近登录时间" sortable="custom"></el-table-column>
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
      users: [],
      page: 1,
      total: 0,
      request: {}
    }
  },
  methods: {
    relpadData() {
      this.$http
        .post(config.service.baseURL + '/dashboard/users', this.request)
        .then(resp => {
          for (var user of resp.data.users) {
            user.updatedAt = util.formatNormalDate(user.updatedAt)
          }
          this.users = resp.data.users
          this.total = resp.data.total
        })
    },
    toDetail(row) {
      this.$router.push('/user/' + row.openid)
    },
    sortChange(order) {
      var d = 0
      if (order.order === 'ascending') {
        d = 1
      } else if (order.order === 'descending') {
        d = -1
      }
      this.request = { page: this.page, orderBy: order.prop, orderDirection: d }
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
