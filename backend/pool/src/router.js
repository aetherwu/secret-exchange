import Vue from 'vue'
import Router from 'vue-router'
import Home from './views/Home.vue'
import Pool from './views/Pool.vue'
import QuestionList from './views/QuestionList.vue'
import Question from './views/Question.vue'
import UserList from './views/UserList.vue'
import User from './views/User.vue'
import ActivityList from './views/ActivityList.vue'
import AnswerList from './views/AnswerList.vue'
import GroupList from './views/GroupList.vue'
import GroupTopicList from './views/GroupTopicList.vue'

Vue.use(Router)

export default new Router({
  base: '/pool/',
  mode: 'history',
  routes: [{
      path: '/',
      name: 'pool',
      component: Pool
    },
    // {
    //   path: '/pool',
    //   name: 'pool',
    //   component: Pool
    // },
    {
      path: '/question/list',
      name: 'QuestionList',
      component: QuestionList
    },
    {
      path: '/question/:id',
      name: 'Question',
      component: Question
    },
    {
      path: '/user/list',
      name: 'UserList',
      component: UserList
    },
    {
      path: '/user/:id',
      name: 'User',
      component: User
    },
    {
      path: '/activity/list',
      name: 'ActivityList',
      component: ActivityList
    },
    {
      path: '/answer/list',
      name: 'AnswerList',
      component: AnswerList
    },
    {
      path: '/group/topics',
      name: 'GroupTopicList',
      component: GroupTopicList
    },
    {
      path: '/group/list',
      name: 'GroupList',
      component: GroupList
    },
  ]
})