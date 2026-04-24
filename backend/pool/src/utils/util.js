function formatTimeStirng(date) {
    return formatTime(new Date(date))
}

function formatTime(date) {
    const current = new Date()
    return dateDiff(date, current)
}

function dateDiff(date, date2) {
    var diff = date.getTime() - date2.getTime()
    diff = Math.abs(diff / 1000)

    const year = date.getFullYear()
    const month = date.getMonth() + 1
    const day = date.getDate()
    const hour = date.getHours()
    const minute = date.getMinutes()
    const second = date.getSeconds()

    if (diff <= 60) {
        return '刚刚'
    } else if (diff < 3600) {
        return Math.ceil(diff / 60) + ' 分钟前'
    } else if (diff < 86400) {
        return Math.ceil(diff / 3600) + ' 小时前'
    } else if (diff < 604800) {
        return Math.ceil(diff / 86400) + ' 天前'
    } else if (year === date2.getFullYear()) {
        return [month, day].map(formatNumber).join('-') + ' ' + [hour, minute, second].map(formatNumber).join(':')
    } else {
        return [year, month, day].map(formatNumber).join('-') + ' ' + [hour, minute, second].map(formatNumber).join(':')
    }
}

function formatNormalDate(d) {
    const date = new Date(d)
    const year = date.getFullYear()
    const month = date.getMonth() + 1
    const day = date.getDate()
    const hour = date.getHours()
    const minute = date.getMinutes()
    const second = date.getSeconds()
    return [year, month, day].map(formatNumber).join('-') + ' ' + [hour, minute, second].map(formatNumber).join(':')
}

function formatNumber(n) {
    n = n.toString()
    return n[1] ? n : '0' + n
}

export default {
    formatTimeStirng: formatTimeStirng,
    formatNormalDate: formatNormalDate
}