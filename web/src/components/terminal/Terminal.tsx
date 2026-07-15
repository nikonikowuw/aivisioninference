// Terminal wraps TerminalView for a cleaner public import path.
// Consumers import from "components/terminal/Terminal" instead of the internal
// TerminalView file, which may also be exported for specialized use cases.
export { default as default } from "./TerminalView";
