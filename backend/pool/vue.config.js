var webpack = require("webpack")

module.exports = {
    configureWebpack: {
        plugins: [
            new webpack.DefinePlugin({
                __DEV__: false
            })
        ]
    },

    baseUrl: undefined,
    outputDir: undefined,
    assetsDir: 'static',
    runtimeCompiler: undefined,
    productionSourceMap: undefined,
    parallel: undefined,
    css: undefined
}