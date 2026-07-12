import {
  Button,
  Center,
  Heading,
  Text,
  VStack,
  useColorModeValue
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

interface NotFoundProps {
  isInner?: boolean;
}

export default function NotFound({ isInner = false }: NotFoundProps) {
  const { t } = useTranslation('common');
  const navigate = useNavigate();
  const textColor = useColorModeValue('navy.700', 'white');
  const mutedColor = useColorModeValue('gray.500', 'gray.400');
  const outerBg = useColorModeValue('gray.50', 'navy.900');

  // 如果是管理内页，去掉全屏的背景和高度
  const bg = isInner ? 'transparent' : outerBg;
  const minH = isInner ? 'calc(100vh - 220px)' : '100vh';

  return (
    <Center minH={minH} bg={bg} px={4}>
      <VStack spacing={6} textAlign="center" py={isInner ? 12 : 0}>
        <Heading
          display="inline-block"
          as="h1"
          fontSize={isInner ? '6xl' : '8xl'}
          bgGradient="linear(to-r, brand.400, brand.600)"
          backgroundClip="text"
          fontWeight="900"
          lineHeight="1"
        >
          404
        </Heading>
        <Text fontSize={isInner ? 'xl' : '2xl'} fontWeight="bold" color={textColor}>
          {t('errors.notFoundTitle') || '页面迷路了'}
        </Text>
        <Text color={mutedColor} maxW="md" fontSize={isInner ? 'xs' : 'sm'}>
          {t('errors.notFoundMessage') || '您访问的页面不存在或已被移除。请检查输入的网址是否正确。'}
        </Text>

        <Button
          colorScheme="brand"
          variant="solid"
          size={isInner ? 'sm' : 'lg'}
          h={isInner ? '40px' : '50px'}
          px={isInner ? '24px' : '32px'}
          onClick={() => navigate('/admin/default')}
          boxShadow="0px 10px 20px rgba(112, 0, 255, 0.2)"
          _hover={{
            transform: 'translateY(-2px)',
            boxShadow: '0px 15px 25px rgba(112, 0, 255, 0.3)',
          }}
          _active={{
            transform: 'translateY(0)',
          }}
          transition="all 0.2s"
        >
          {t('errors.backHome') || '返回首页'}
        </Button>
      </VStack>
    </Center>
  );
}
