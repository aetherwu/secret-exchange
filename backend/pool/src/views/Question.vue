<template>
    <div>
        <Nav></Nav>
        <div v-if="question">
            <div>问题：{{question.question.content}}</div>
            <el-table :data="question.answers" @cell-click="toUser">
                <el-table-column label="头像" width="80">
                    <template slot-scope="scope">
                        <img class="avatar" :src="scope.row.user.avatar" />
                    </template>
                </el-table-column>
                <el-table-column prop="user.nickname" label="昵称" width="150" @click="toUser"></el-table-column>
                <el-table-column prop="content" label="答案"></el-table-column>
                <el-table-column prop="createdAt" width="150" label="回答时间"></el-table-column>
                <el-table-column label="操作" width="100">
                    <template slot-scope="scope">
                        <el-button @click="editAnswer(scope.row)" type="text" size="small">编辑</el-button>
                        <el-button @click="deleteAnswer(scope.row)" type="text" size="small">删除</el-button>
                    </template>
                </el-table-column>
            </el-table>
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
      id: '',
      question: null
    }
  },
  methods: {
    relpadData() {
      this.$http
        .post(config.service.baseURL + '/dashboard/questions/detail', {
          id: this.id
        })
        .then(resp => {
          for (var q of resp.data.answers) {
            q.createdAt = util.formatTimeStirng(q.createdAt)
          }
          this.question = resp.data
        })
    },
    editAnswer(e) {
      this.$prompt('请输入答案', '提示', {
        inputValue: e.content,
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })
        .then(({ value }) => {
          if (value && value.length > 0) {
            this.onEditAnswer({ id: e.id, content: value })
          } else {
            this.$message({
              type: 'info',
              message: '请输入答案'
            })
          }
        })
        .catch(() => {})
    },
    onEditAnswer(data) {
      this.$http
        .post(config.service.baseURL + '/dashboard/answers/edit', data)
        .then(resp => {
          this.relpadData()
        })
    },
    deleteAnswer(e) {
      this.$confirm('删除该答案, 是否继续?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
        .then(() => {
          this.$http
            .post(config.service.baseURL + '/dashboard/answers/delete', {
              id: e.id
            })
            .then(resp => {
              this.relpadData()
            })
        })
        .catch(() => {})
    },
    toUser(row, column) {
      if (column.label === '昵称' || column.label === '头像') {
          this.$router.push('/user/' + row.user.openID)
      }
    }
  },
  created() {
    this.id = this.$route.params.id
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
