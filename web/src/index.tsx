import './assets/css/App.css';
import './i18n';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';
import App from './App';

// 拦截 console.error，防止 React DevTools (installHook.js) 格式化报错时崩溃
// 如果 DevTools 崩溃，它会抛出错误，导致 React 误以为是组件渲染错误并掩盖原始错误
const originalConsoleError = console.error;
console.error = function (...args) {
  try {
    originalConsoleError.apply(console, args);
  } catch (e) {
    // 如果原始 console.error（被 DevTools 劫持后）崩溃了，我们用 warn 打印真实参数
    console.warn('console.error crashed (likely DevTools bug). Original args:', args);
  }
};

const root = createRoot(document.getElementById('root')!);

root.render(
  <BrowserRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
    <App />
  </BrowserRouter>,
);
