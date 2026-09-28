// Mobile Web 页面内联脚本「真实 Script 语法」回归验证
//
// 背景：Change 001 曾出现「Go 单测全绿 + DOM shim 全绿，但手机浏览器整段 JS 失效」的
// 缺陷——根因是 classic <script> 顶层 return，浏览器报 parse 期 SyntaxError:
// Illegal return statement，事件没注入前脚本就被拒绝执行。
//
// 前两种护栏都看不见它：
//   1. go test            → //go:embed 的静态页只核 200/Content-Type/固定文本，JS 在进程外
//   2. new Function(shim) → 函数体包裹令顶层 return 变成「合法 return」（同源假阴性）
//   * 连 node --check 也看不见它：.js 按 CommonJS 模块语义解析，模块本就包在函数里，
//     顶层 return 合法，--check 返回 0。
//
// 因此这里固定用 node:vm 的 Script 目标编译——这正是浏览器对 classic <script> 的解析
// 语义（Sloppy 全局脚本，不允许顶层 return）。只要 index.html 内联脚本有一个顶层 return，
// 这里就抛 SyntaxError，退出码非 0，绝无假阴性。
//
// 用法：node _dev/mobile-check/check-raw-syntax.js
// 退出码：0 = 语法通过；非 0 = 存在真实 Script 语法错误。

'use strict';

const fs = require('fs');
const path = require('path');
const vm = require('vm');

const htmlPath = path.resolve(__dirname, '../../pkg/mobile/web/index.html');
const html = fs.readFileSync(htmlPath, 'utf8');
const match = html.match(/<script>([\s\S]*?)<\/script>/);
if (!match) {
  console.error('FAIL：未找到 <script> 块');
  process.exit(1);
}
const inlineJs = match[1];

// 按浏览器 classic <script> 语义做 Script 目标编译。
// 不套函数、不套 IIFE、不 new Function —— 这三点正是之前漏检的原因。
try {
  new vm.Script(inlineJs, { filename: 'web/index.html inline <script>' });
} catch (e) {
  console.error(`✗ 内联脚本存在真实 Script 语法错误：`);
  console.error(`  ${e.constructor.name}: ${e.message}`);
  process.exit(1);
}

console.log('✓ 内联脚本通过真实 Script 目标语法检查（顶层无非法 return）');
process.exit(0);