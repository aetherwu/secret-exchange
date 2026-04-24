<template>
  <div>
    <Nav></Nav>
    <el-table  :data="activitys" @cell-click="pushTo" @sort-change="sortChange" :default-sort="{prop: 'createdAt', order: 'descending'}" stripe>
      <el-table-column prop="question.content" label="问题"></el-table-column>
      <el-table-column prop="source.user.nickname" label="被交换人" width="150"></el-table-column>
      <el-table-column label="被交换答案">

        <template slot-scope="scope">
        <span style="margin-left: 10px">{{ scope.row.source.content }}</span>
        </template>

      </el-table-column>
      <el-table-column prop="swap.user.nickname" label="交换人" width="150"></el-table-column>
      <el-table-column prop="swap.content" label="交换答案"></el-table-column>
      <el-table-column prop="createdAt" label="交换时间" sortable="custom"></el-table-column>
    </el-table>

    <el-table :data="tableData">
      <el-table-column
        v-for="{ prop, label } in colConfigs"
        :key="prop"
        :prop="prop"
        :label="label">
      </el-table-column>
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
      activitys: [],
      page: 1,
      total: 0,
      request: {}
    }
  },
  methods: {
    reloadData() {
      this.$http
        .post(config.service.baseURL + '/dashboard/activitys', this.request)
        .then(resp => {
          for (var a of resp.data.activitys) {
            a.createdAt = util.formatTimeStirng(a.createdAt)
          }
          this.activitys = resp.data.activitys
          this.total = resp.data.total
        })
    },
    pushTo(row, column) {
      if (column.label === '问题') {
        this.$router.push('/question/' + row.question.id)
      } else if (column.label === '被交换人') {
        this.$router.push('/user/' + row.source.user.openID)
      } else if (column.label === '交换人') {
        this.$router.push('/user/' + row.swap.user.openID)
      }
    },
    sortChange(order) {
      var d = 0
      if (order.order === 'ascending') {
        d = 1
      } else if (order.order === 'descending') {
        d = -1
      }
      this.request = { page: this.page, orderBy: order.prop, orderDirection: d }
      this.reloadData()
    },
    pageChange(page) {
      this.page = page
      this.request.page = page
      this.reloadData()
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
