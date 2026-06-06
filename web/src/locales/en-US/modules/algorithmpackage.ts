export const algorithmpackage = {
  "title": "Algorithm Packages",
  "uploadZone": {
    "title": "Click or drag ZIP algorithm package to this area to upload",
    "hint": "Supports packages up to 1GB. The system will automatically repack (ZIP -> TAR) and execute the self-test.",
    "onlyZip": "Only ZIP files are supported.",
    "dropHere": "Drop here..."
  },
  "uploading": "Uploading...",
  "checking": "Self-checking...",
  "card": {
    "version": "Version",
    "domain": "Domain",
    "hardware": "Hardware Platform",
    "checkStatus": "Self-check Status",
    "packageSize": "Package Size",
    "md5": "MD5 Checksum",
    "capabilities": "Capabilities",
    "details": "Details",
    "delete": "Delete Package",
    "deleteConfirm": "Are you sure you want to delete this algorithm package? This will permanently remove files and cannot be undone.",
    "selfCheckBtn": "Run Selfcheck",
    "selfCheckSuccess": "Self-check Passed",
    "selfCheckFailed": "Self-check Failed",
    "selfCheckRunning": "Self-check Running",
    "selfCheckPending": "Self-check Pending",
    "schema": "Result Schema",
    "paramsSchema": "Parameters Schema",
    "copied": "Copied to clipboard",
    "noSchema": "No schema provided",
    "errorMsg": "Self-check Error",
    "empty": "No algorithm packages yet. Upload one above."
  },
  "message": {
    "uploadSuccess": "Algorithm package uploaded successfully, self-check triggered",
    "uploadFailed": "Failed to upload package",
    "deleteSuccess": "Algorithm package deleted successfully",
    "deleteFailed": "Failed to delete package",
    "selfCheckTriggered": "Self-check command sent successfully",
    "selfCheckFailed": "Failed to send self-check command",
    "operationFailed": "Operation failed"
  },
  "search": {
    "placeholder": "Search algorithm name, alias, or domain..."
  },
  "stats": {
    "total": "Total Packages",
    "passed": "Self-check Passed",
    "failed": "Self-check Failed",
    "active": "Active"
  }
} as const;

export default algorithmpackage;
