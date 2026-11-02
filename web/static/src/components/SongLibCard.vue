<template>
<section class="songlib-card">
  <div class="songlib-product"><img src="/songlib-mark.png" width="64" height="64" alt="音屿"><div><h2>让散落的音乐，回到自己的岛屿</h2><p>音屿播放器 · 一次购买最多激活 6 台，iOS / Apple TV / Mac 共享（Mac 版推出后适用）。</p></div></div>
  <nav><a :href="links.download" target="_blank" rel="noopener noreferrer">下载 Apple TV 版</a><a :href="links.ios" target="_blank" rel="noopener noreferrer">了解 iOS 版</a><span>Mac 版即将推出</span><a :href="links.trial" target="_blank" rel="noopener noreferrer">领取 30 天 Pro 体验</a><a v-if="links.basic" :href="links.basic" target="_blank" rel="noopener noreferrer">购买基础版</a><a v-if="links.pro" :href="links.pro" target="_blank" rel="noopener noreferrer">购买 Pro</a><span v-if="!links.basic && !links.pro">购买通道尚未配置</span></nav>
  <details><summary>首次自签前，请先阅读账户与续签说明</summary><p>请专门注册用于自签的 Apple 账户小号，启用 2FA 并保存恢复方式；不要使用存放照片、支付与个人资料的主账号。不需要退出电视当前主用账号。不支持 App 专用密码。</p><p>免费个人账户的签名通常 7 天到期。NAS 断网、Apple TV 离线、账户风控或二次认证可能导致续签失败，请查看应用状态并手动重试。自动续签长期稳定性待观察。</p><p>Apple 签名到期与音屿商业授权/体验到期分别管理。安装助手本体永久免费，导入其他合法 IPA 不需要购买音屿。</p><p><a href="/source">下载本版本完整可构建源码</a> · <a href="/license">AGPL-3.0 开源许可</a> · <a href="https://github.com/bitxeno/atvloadly" target="_blank" rel="noopener noreferrer">ATVLoadly 上游</a></p></details>
</section>
</template>
<script setup>
import {reactive,onMounted} from 'vue';
const links=reactive({download:'https://songlib.playsong.cn/#download-tv',ios:'https://songlib.playsong.cn/#download-ios',trial:'https://songlib.playsong.cn/#trial',basic:null,pro:null});
onMounted(async()=>{try{const r=await fetch('/songlib/config');if(r.ok)Object.assign(links,await r.json())}catch{}});
</script>
<style scoped>
.songlib-card{margin:0 0 28px;padding:24px;border:1px solid #e2c68c55;border-radius:16px;background:#202923;color:#f4f0e8}.songlib-product{display:flex;gap:18px;align-items:center}.songlib-product img{object-fit:contain;flex-shrink:0}h2{font-size:20px}p{margin:10px 0;line-height:1.7;color:#cbd2ca}nav{display:flex;flex-wrap:wrap;gap:10px;margin:18px 0}nav a,nav span{padding:8px 12px;border:1px solid #e2c68c55;border-radius:8px;font-size:14px}a{color:#e2c68c}summary{cursor:pointer}nav span{color:#a2b0a5}@media(max-width:600px){.songlib-card{padding:16px}.songlib-product{align-items:flex-start}h2{font-size:18px}}
</style>
