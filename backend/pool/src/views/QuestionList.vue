<template>
  <div class="pool">
    <Nav />

    <el-button type="primary" @click="inputQuestion">添加问题</el-button>

    <div class="margin" style="display: flex;align-items: center;justify-content: center;">
      <span style="margin-right: 15px">Lv: </span>
      <el-radio-group v-model="filterLevelValue" @change="filterLevelChange" size="small">
        <el-radio-button label="All"></el-radio-button>
        <el-radio-button label="0"></el-radio-button>
        <el-radio-button label="1"></el-radio-button>
        <el-radio-button label="2"></el-radio-button>
        <el-radio-button label="3"></el-radio-button>
        <el-radio-button label="4"></el-radio-button>
      </el-radio-group>
      <div class="find-input">
        <el-input v-model="findInput" placeholder="请输入内容" clearable @input="onFindQuestions">
          <el-button slot="append" icon="el-icon-search" @click="onFindQuestions"></el-button>
        </el-input>
      </div>
    </div>

    <el-table :data="questions" ref="multipleTable" @cell-click="toDetail" @sort-change="sortChange" :default-sort="{prop: 'createdAt', order: 'descending'}" @selection-change="handleSelectionChange" stripe>
      <el-table-column v-if="showSelect" type="selection" width="100"></el-table-column>
      <el-table-column prop="content" label="问题"></el-table-column>
      <el-table-column prop="level" label="等级" width="100" sortable="custom"></el-table-column>
      <el-table-column prop="answerCount" label="回答次数" width="100" sortable="custom"></el-table-column>
      <el-table-column prop="createdAt" label="添加时间" width="150" sortable="custom"></el-table-column>
      <el-table-column label="操作" width="100">
        <template slot-scope="scope">
          <el-button @click.stop="editQuestion(scope.row)" type="text" size="small">编辑</el-button>
          <el-button @click.stop="deleteQuestion(scope.row)" type="text" size="small">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="footer">
      <div class="footer-left">
        <el-button type="primary" @click="onShowSelect">{{showSelect ? '取消设置':'设置等级'}}</el-button>
        <div v-if="showSelect" style="margin-top: 20px;">
          <el-select v-model="levelValue" placeholder="请选择等级">
            <el-option v-for="item in levelOptions" :key="item.value" :label="item.label" :value="item.value">
            </el-option>
          </el-select>
          <el-button style="margin-top: 20px;" type="primary" @click="onEditManyLevel">确定修改</el-button>
        </div>
      </div>
      <div class="footer-right">
        <el-pagination layout="prev, pager, next" :total="total" :page-size="200" @current-change="pageChange"></el-pagination>
      </div>
    </div>
  </div>
</template>

<script>
import config from '../config'
import Nav from '@/components/Nav'
import util from '../utils/util'

export default {
  components: {
    Nav
  },
  data() {
    return {
      questions: [],
      page: 1,
      total: 0,
      request: {},
      findInput: '',
      multipleSelection: [],
      showSelect: false,
      levelOptions: [
        {
          value: '0',
          label: '0'
        },
        {
          value: '1',
          label: '1'
        },
        {
          value: '2',
          label: '2'
        },
        {
          value: '3',
          label: '3'
        },
        {
          value: '4',
          label: '4'
        }
      ],
      levelValue: '',
      filterLevelValue: 'All'
    }
  },
  methods: {
    inputQuestion() {
      this.$prompt('请输入问题', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })
        .then(({ value }) => {
          if (value && value.length > 0) {
            this.addQuestion(value)
          } else {
            this.$message({
              type: 'info',
              message: '请输入问题'
            })
          }
        })
        .catch(() => {})
    },
    addQuestion(q) {
      this.$http
        .post(config.service.baseURL + '/dashboard/questions/add', {
          content: q
        })
        .then(resp => {
          this.loadQuestions()
        })
        .catch(err => {
          console.log(err)
        })
    },
    loadQuestions() {
      var url = config.service.baseURL + '/dashboard/questions'
      if (this.findInput.length > 0) {
        url = config.service.baseURL + '/dashboard/questions/find'
      }
      this.$http
        .post(url, this.request)
        .then(resp => {
          for (var item of resp.data.questions) {
            item.createdAt = util.formatNormalDate(item.createdAt)
          }
          this.questions = resp.data.questions
          this.total = resp.data.total
        })
        .catch(err => {
          console.log(err)
        })
    },
    deleteQuestion(e) {
      this.$confirm('删除该问题, 是否继续?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })
        .then(() => {
          this.$http
            .post(config.service.baseURL + '/dashboard/questions/remove', {
              id: e.id
            })
            .then(resp => {
              this.loadQuestions()
            })
        })
        .catch(() => {})
    },
    editQuestion(e) {
      this.$prompt('请输入问题', '提示', {
        inputValue: e.content,
        confirmButtonText: '确定',
        cancelButtonText: '取消'
      })
        .then(({ value }) => {
          if (value && value.length > 0) {
            this.onEditQuestion({ id: e.id, content: value })
          } else {
            this.$message({
              type: 'info',
              message: '请输入问题'
            })
          }
        })
        .catch(() => {})
    },
    onEditQuestion(data) {
      this.$http
        .post(config.service.baseURL + '/dashboard/questions/edit', data)
        .then(resp => {
          this.loadQuestions({})
        })
        .catch(err => {
          console.log(err)
        })
    },
    toDetail(row, column) {
      if (column.label === '问题') {
        this.$router.push('/question/' + row.id)
      }
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
    onFindQuestions(e) {
      this.request = {}
      this.filterLevelValue = 'All'
      if (this.findInput.length === 0) {
        this.loadQuestions()
      } else {
        this.request.content = this.findInput
        this.loadQuestions()
      }
    },
    handleSelectionChange(val) {
      this.multipleSelection = val
    },
    onShowSelect() {
      this.showSelect = !this.showSelect
      if (!this.showSelect) {
        this.levelValue = ''
        this.$refs.multipleTable.clearSelection()
        this.multipleSelection = []
      }
    },
    onEditManyLevel() {
      if (this.levelValue === '') {
        this.$message({
          message: '请选择等级',
          type: 'warning'
        })
        return
      }

      if (this.multipleSelection.length === 0) {
        this.$message({
          message: '请选择问题',
          type: 'warning'
        })
        return
      }

      var ids = this.multipleSelection.map(e => {
        return e.id
      })

      this.$http
        .post(config.service.baseURL + '/dashboard/questions/edit/level', {
          questions: ids,
          level: parseInt(this.levelValue)
        })
        .then(resp => {
          this.loadQuestions()
        })
        .catch(err => {
          console.log(err)
        })
    },
    filterLevelChange(e) {
      if (e === 'All') {
        this.request.filterBy = ''
        this.request.filterValue = 0
      } else {
        this.request.filterBy = 'level'
        this.request.filterValue = parseInt(e)
      }
      this.loadQuestions()
    }
  }
}
</script>


<style type="text/css">
.find-input {
  width: 200px;
  margin-left: 50px;
}

.margin {
  margin: 40px 0;
}

.footer {
  padding: 40px 0;
  display: flex;
  align-items: center;
  justify-content: flex-start;
}

.footer-left {
  width: 200px;
  display: flex;
  align-items: center;
  flex-direction: column;
}

.footer-right {
  margin-left: 100px;
}
</style>