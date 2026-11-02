let sessionPromise;
export function managementSession(){
  if(!sessionPromise)sessionPromise=fetch('/management/session').then(async r=>{if(!r.ok){location.href='/management/login';throw new Error('管理会话已过期')}return r.json()}).catch(e=>{sessionPromise=null;throw e});
  return sessionPromise;
}
