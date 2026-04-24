import Vue from 'vue'
import App from './App.vue'
import router from './router'
import http from './http'
import {
  Button,
  Table,
  TableColumn,
  Input,
  Pagination,
  Message,
  MessageBox,
  RadioGroup,
  RadioButton,
  Select,
  Option
} from 'element-ui';


Vue.config.productionTip = false

Vue.use(http);

Vue.use(Button);
Vue.use(Table);
Vue.use(TableColumn);
Vue.use(Input);
Vue.use(Pagination);
Vue.use(RadioGroup);
Vue.use(RadioButton);
Vue.use(Select);
Vue.use(Option);

Vue.prototype.$prompt = MessageBox.prompt;
Vue.prototype.$message = Message;



new Vue({
  router,
  render: h => h(App)
}).$mount('#app')