const PREFIX = "trainingops:upload-resume:";

export function buildUploadResumeKey(
  tenantID: string,
  userID: string,
  fileName: string,
  fileSize: number,
  documentID: string,
): string {
  return `${PREFIX}${tenantID}:${userID}:${fileName}:${fileSize}:${documentID || "new"}`;
}

export function clearUploadResumeCache(tenantID?: string, userID?: string) {
  const scopedPrefix =
    tenantID && userID ? `${PREFIX}${tenantID}:${userID}:` : PREFIX;
  const remove: string[] = [];
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i);
    if (!key) {
      continue;
    }
    if (key.startsWith(scopedPrefix)) {
      remove.push(key);
    }
  }
  for (const key of remove) {
    localStorage.removeItem(key);
  }
}
