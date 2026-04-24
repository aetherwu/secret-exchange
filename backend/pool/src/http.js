import Vue from 'vue'
import axios from 'axios'
import {
    Message
} from 'element-ui'

const Axios = axios.create({

})


Axios.interceptors.request.use(config => {
    //   var token = store.state.users[store.state.account]
    //   if (token && token.length > 0) {
    //     config.headers.Authorization = token
    //   }
    return config
}, error => {
    Message.error(error.message)
    return Promise.reject(error.message)
})

Axios.interceptors.response.use(resp => {
    if (resp.data.errCode != null && resp.data.errCode !== 0) {
        Message.error(resp.data.errMsg)
        return Promise.reject(resp.data.errMsg)
    }
    return resp
}, error => {
    Message.error(error.message)
    return Promise.reject(error.message)
})

export default {
    install: function (Vue, options) {
        Object.defineProperty(Vue.prototype, '$http', {
            value: Axios
        })
    }
}