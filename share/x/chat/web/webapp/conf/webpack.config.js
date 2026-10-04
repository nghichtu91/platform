const webpack = require('atool-build/lib/webpack')
const HtmlWebpackPlugin = require('html-webpack-plugin')
const {CleanWebpackPlugin} = require('clean-webpack-plugin')
const path = require('path')

module.exports = function (webpackConfig, env) {
    webpackConfig.babel.plugins.push('transform-runtime')
    webpackConfig.babel.plugins.push(['import', {
        libraryName: 'antd',
        style: true
    }])

    webpackConfig.devtool = '#eval' //#inline-source-map

    // Support hmr
    if (env === 'development') {
        webpackConfig.babel.plugins.push(['dva-hmr', {
            entries: [
                './src/index.js'
            ]
        }])
    } else {
        webpackConfig.babel.plugins.push('dev-expression')
        webpackConfig.entry = {
            index: './src/index.js',
            // common: [ 'react', 'react-dom', 'classnames', 'antd', 'dva', 'dva-loading', 'qs', 'js-cookie', 'moment', 'rc-queue-anim', 'rc-tween-one']
        }
    }
    //mock data config
    webpackConfig.plugins.push(new webpack.DefinePlugin({
        'taiyouxi.app.admin.ISMOCK': false,
        'taiyouxi.app.admin.IS_DYNAMIC_LOAD': true,
        'taiyouxi.app.admin.API_HOST': JSON.stringify(''),
        'taiyouxi.app.admin.SOCKET_HOST': JSON.stringify('ws://127.0.0.1:3000'),
        'taiyouxi.app.admin.CLIENT_ID': JSON.stringify('gmtool.taiyouxi.cn'),
        'taiyouxi.app.admin.CLIENT_SECRET': JSON.stringify('st4mtjULIuh2ks6r'),
        'taiyouxi.app.admin.GRANT_TYPE': JSON.stringify('client_credentials')
    }))

    // Don't extract common.js and common.css (please extract if use common.js)
    webpackConfig.plugins = webpackConfig.plugins.filter(function (plugin) {
        return !(plugin instanceof webpack.optimize.CommonsChunkPlugin)
    })

    webpackConfig.plugins.push(new HtmlWebpackPlugin({
        template: 'src/index.ejs', // 源模板文件
        filename: './index.html', // 输出文件【注意：这里的根路径是module.exports.output.path】
        showErrors: true,
        inject: 'body',
        chunks: ["common", 'index']

    }));
    webpackConfig.plugins.push(new CleanWebpackPlugin());
    // Support CSS Modules
    // Parse all less files as css module.
    webpackConfig.module.loaders.forEach(function (loader, index) {
        if (typeof loader.test === 'function' && loader.test.toString().indexOf('\\.less$') > -1) {
            loader.include = /node_modules/
            loader.test = /\.less$/
        }
        if (loader.test.toString() === '/\\.module\\.less$/') {
            loader.exclude = /node_modules/
            loader.test = /\.less$/
        }
        if (typeof loader.test === 'function' && loader.test.toString().indexOf('\\.css$') > -1) {
            loader.include = /node_modules/
            loader.test = /\.css$/
        }
        if (loader.test.toString() === '/\\.module\\.css$/') {
            loader.exclude = /node_modules/
            loader.test = /\.css$/
        }
    })

    webpackConfig.resolve.alias = {
        '@': path.join(__dirname, '..', 'src')
    }

    return webpackConfig
}
