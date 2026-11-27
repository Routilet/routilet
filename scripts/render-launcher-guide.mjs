import fs from "node:fs/promises";
const [source,output,companion,launcher,mode] = process.argv.slice(2);
// Normalize checkout line endings before exact template replacements.
let html=(await fs.readFile(source,"utf8")).replace(/\r\n/g,"\n");
function replace(from,to) {
  if (!html.includes(from)) throw Error("Guide source has changed: "+from.slice(0,60));
  html=html.replace(from,()=>to);
}
if (mode==="1") {
  const preview=launcher.includes("-");
  replace("从第一次导入比赛，到使用 C Exam Kit 写出程序、测试并提交。","从第一次导入比赛，到使用 C Exam Kit 写程序、测试、观察并提交。"+(preview?"当前为 v2 预览审核稿，尚非正式说明。":""));
  replace("这两个工具是做什么的？","这些工具是做什么的？");
  replace("<dt>C / C++ 编译器</dt>","<dt>C Observer</dt><dd>记录 C 程序的执行历史，帮助查看变量变化、数组与输出。它不生成解题代码。</dd>\n      <dt>C / C++ 编译器</dt>");
  replace("保持联网，选择「安装」，启动器会自动准备运行环境、Companion 和 C Exam Kit。安装就绪后选择「启动」。",(preview?"保持联网。预览审核时先进入「设置 → 切换发行渠道」选择预览版，然后选择「安装」。":"保持联网，保持默认稳定版渠道，选择「安装」。")+"启动器暂存完整软件，展示实际组件和版本，确认后启用。安装就绪后选择「启动」。稳定版与预览版安装、配置和凭据分离。");
  replace("› 安装\n  使用说明</pre>","› 安装\n  使用说明\n  设置</pre>");
  replace("<a href=\"#update\">5. 日常使用</a>","<a href=\"#observe\">5. 程序观察</a><a href=\"#update\">6. 日常使用</a>");
  replace('<section id="update">',`<section id="observe">
  <h2>观察程序的执行过程</h2>
  <article><h3>从样例进入观察器</h3>
    <p>保存 main.c 后，在题目菜单选择「运行观察」，或者在样例详情按 O。选择输入后，观察器在本机编译并运行程序，采集有限的执行历史。它不会改写源码，也不会向 OJ 提交观察用源码。</p>
    <p>Windows x64 使用 GCC/GDB；macOS 使用 Clang/LLDB；Linux 根据可用工具选择。缺少调试器时参照编译环境配置指引。正常本地评测不需要 GDB 或 LLDB。</p>
  </article>
  <article><h3>查看执行历史，而不是只看答案</h3>
    <p>左右方向键回放快照，Home/End 到首尾，J 跳转，? 查看操作帮助，Q 退出。观察变量值、数组元素及源码所在行，重点看与自己预期不同的变化。记录的是采集到的快照，不是完整赋值日志。</p>
    <p>输出对照保留原始字节；支持时另显示按样例规则的一致性。退出后回到 Companion 原页面。观察结果不是 OJ 判定，考试能否使用以课程规定为准。</p>
    ${preview?'<p class="tip">预览审核稿：观察器实际截图将在实机审核后补充。</p>':""}
  </article>
</section>
<section id="update">`);
  replace("启动器会检查 Companion 和 C Exam Kit 的更新。没有更新提示时，直接选「启动」。旧版用户需重新下载 Routilet 新启动器；它仍可离线启动已安装的旧软件，两个新发行包均可用时才提示迁移。迁移不会删除课程文件、资源快照或登录状态。","启动器统一管理 Companion 及兼容依赖。选择「更新」后先暂存安装，确认实际组件变化才启用；取消不影响原安装。已有旧启动器安装可在稳定渠道离线启动，迁移不会删除课程、资源快照或登录状态。预览软件不继承稳定版的登录与课程状态。");
  replace("资源下载后，启动器可预览当前课程的升级范围；也可在 Companion 的「设置 → 更新课程资源」中处理。","资源下载后，在 Companion 的「设置 → 更新课程资源」中处理。");
  replace("</article>\n</section>\n<footer>",`</article>
  <article><h3>回退、切换渠道及卸载</h3>
    <p>设置中可以切换稳定版与预览版；有上一份安装时可以离线回退。回退软件不等于回退课程。卸载当前新版渠道不会删除课程、账号配置、旧版安装或另一渠道。</p>
    <p>联网失败时仍可启动已经安装的版本。${preview?"预览审核使用 next 渠道，请勿把预览 ZIP 当作正式发行转发。":"日常使用保持稳定版渠道；只有试用预发行版本时才切换至预览版。"}</p>
  </article>
</section>
<footer>`);
}
html=html.replaceAll("@VERSION@",companion).replaceAll("@LAUNCHER_VERSION@",launcher);
await fs.writeFile(output,html);
