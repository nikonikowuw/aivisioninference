import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Box, Button, Center, Flex, Heading, Text, useColorModeValue } from '@chakra-ui/react';
import { MdErrorOutline, MdRefresh } from 'react-icons/md';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

/**
 * 通用错误边界组件
 *
 * 捕获子组件树中的渲染错误，显示友好的错误提示而非空白页面。
 * 可用于路由级别、模块级别或组件级别的错误隔离。
 *
 * @example
 * ```tsx
 * <ErrorBoundary>
 *   <LazyComponent />
 * </ErrorBoundary>
 * ```
 */
export default class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // 调用外部 onError 回调（如日志上报）
    this.props.onError?.(error, errorInfo);

    // 使用 console.warn 绕过 DevTools 对 console.error 的劫持崩溃
    // eslint-disable-next-line no-console
    console.warn(`[ErrorBoundary] ====== Component Error Details ======`);
    // eslint-disable-next-line no-console
    console.warn(`[ErrorBoundary] Message:`, error?.message || error);
    // eslint-disable-next-line no-console
    console.warn(`[ErrorBoundary] Name:`, error?.name);
    // eslint-disable-next-line no-console
    console.warn(`[ErrorBoundary] Stack:`, error?.stack);
    // eslint-disable-next-line no-console
    console.warn(`[ErrorBoundary] Component Stack:`, errorInfo?.componentStack);
    // eslint-disable-next-line no-console
    console.warn(`[ErrorBoundary] ===================================`);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  handleViewDetail = () => {
    if (this.state.error) {
      // eslint-disable-next-line no-console
      console.warn(`[ErrorBoundary] Full error detail:`, this.state.error);
    }
  };

  render() {
    if (this.state.hasError) {
      // 如果传入了自定义 fallback，优先使用
      if (this.props.fallback) {
        return this.props.fallback;
      }

      // 默认错误提示
      return <DefaultFallback error={this.state.error} onReset={this.handleReset} />;
    }

    return this.props.children;
  }
}

/**
 * 默认错误回退 UI
 */
function DefaultFallback({ error, onReset }: { error: Error | null; onReset: () => void }) {
  const bgColor = useColorModeValue('white', 'navy.800');
  const textColor = useColorModeValue('navy.700', 'white');
  const secondaryTextColor = useColorModeValue('gray.500', 'gray.400');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  return (
    <Center h="100vh" p="20px">
      <Box
        bg={bgColor}
        p="40px"
        borderRadius="20px"
        border="1px solid"
        borderColor={borderColor}
        textAlign="center"
        maxW="500px"
        w="100%"
      >
        <Flex direction="column" align="center" gap="4">
          <Box fontSize="64px" color="red.400" mb="8px">
            <MdErrorOutline />
          </Box>
          <Heading as="h2" size="lg" color={textColor} mb="8px">
            页面加载异常
          </Heading>
          <Text color={secondaryTextColor} mb="24px" fontSize="sm">
            渲染时发生了意外错误，请尝试刷新页面。
            {error?.message && (
              <>
                <br />
                错误信息：{error.message}
              </>
            )}
          </Text>
          <Button
            leftIcon={<MdRefresh />}
            colorScheme="brand"
            onClick={onReset}
          >
            重新加载
          </Button>
        </Flex>
      </Box>
    </Center>
  );
}

/**
 * 路由级别错误边界
 * 包裹在 Suspense 内部，只捕获目标组件的渲染错误
 */
export function RouteErrorFallback() {
  const bgColor = useColorModeValue('white', 'navy.800');
  const textColor = useColorModeValue('navy.700', 'white');
  const secondaryTextColor = useColorModeValue('gray.500', 'gray.400');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  return (
    <Center h="400px" p="20px">
      <Box
        bg={bgColor}
        p="32px"
        borderRadius="16px"
        border="1px solid"
        borderColor={borderColor}
        textAlign="center"
        maxW="400px"
        w="100%"
      >
        <Box fontSize="48px" color="red.400" mb="12px">
          <MdErrorOutline />
        </Box>
        <Text fontSize="lg" fontWeight="600" color={textColor} mb="8px">
          页面组件加载失败
        </Text>
        <Text fontSize="sm" color={secondaryTextColor}>
          请刷新页面或联系管理员
        </Text>
      </Box>
    </Center>
  );
}
