let sessionPromise;

export function managementSession() {
  if (!sessionPromise) {
    sessionPromise = fetch("/management/session")
      .then(async (r) => {
        if (!r.ok) {
          location.href = "/management/login";
          throw new Error("管理会话已过期");
        }
        return r.json();
      })
      .catch((e) => {
        sessionPromise = null;
        throw e;
      });
  }
  return sessionPromise;
}

export function updateManagementCSRF(csrf) {
  if (typeof csrf === "string" && csrf) {
    sessionPromise = Promise.resolve({ csrf });
  } else {
    sessionPromise = null;
  }
}

export async function logoutManagement() {
  try {
    const { csrf } = await managementSession();
    await fetch("/management/logout", {
      method: "POST",
      headers: { "X-CSRF-Token": csrf },
    });
  } finally {
    sessionPromise = null;
    location.href = "/management/login";
  }
}
