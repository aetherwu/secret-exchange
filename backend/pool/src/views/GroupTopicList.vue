<template>
    <div>
        <Nav></Nav>
        <el-table :data="groupTopics" ref="multipleTable" @sort-change="sortChange" :default-sort="{prop: 'createdAt', order: 'descending'}" stripe>
            <el-table-column prop="groupID" label="群ID" width="300" ></el-table-column>
            <el-table-column prop="content" label="问题"></el-table-column>
            <el-table-column prop="createdAt" label="创建时间" width="150" sortable="custom"></el-table-column>
            <el-table-column prop="addUsers.length" label="参与人数" width="150" sortable="custom"></el-table-column>

            
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
      groupTopics: [],
      page: 1,
      total: 0,
    }
  },
  methods: {
    loadQuestions() {
      var url = config.service.baseURL + '/dashboard/group/topics'
      this.$http
        .post(url, this.request)
        .then(resp => {
          for (var item of resp.data.groupTopics) {
            item.createdAt = util.formatNormalDate(item.createdAt)
          }
          this.groupTopics = resp.data.groupTopics
          this.total = resp.data.total
        })
        .catch(err => {
          console.log(err)
        })
    },
    sortChange(order) {
      var d = 0
      if (order.order === 'ascending') {
        d = 1
      } else if (order.order === 'descending') {
        d = -1
      }
      this.request = {
        page: this.page,
        orderBy: order.prop,
        orderDirection: d,
        content: this.findInput
      }
      this.loadQuestions()
    },
    pageChange(page) {
      this.page = page
      this.request.page = page
      this.loadQuestions()
    },
  },
}
</script>

<style scoped>
.avatar {
  width: 50px;
  height: 50px;
}
</style>
