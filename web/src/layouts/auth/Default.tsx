import type { ReactNode } from 'react';
import { Box, Flex, Text, Heading, useColorModeValue, VStack, Icon } from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import FixedPlugin from 'components/fixedPlugin/FixedPlugin';
import LanguageSwitcher from 'components/i18n/LanguageSwitcher';
import { MdOutlineDashboard, MdOutlineSecurity, MdOutlineSpeed } from 'react-icons/md';

function AuthIllustration(props: { children: ReactNode }) {
  const { children } = props;
  const { t } = useTranslation('auth');
  
  // Dynamic colors
  const bgGradient = useColorModeValue(
    'linear-gradient(135deg, #422AFB 0%, #3182CE 100%)',
    'linear-gradient(135deg, #11047A 0%, #0B1437 100%)'
  );
  
  const formBg = useColorModeValue('white', 'navy.900');
  const textColor = 'white';
  const glassBg = useColorModeValue('rgba(255, 255, 255, 0.1)', 'rgba(11, 20, 55, 0.2)');
  const glassBorder = useColorModeValue('rgba(255, 255, 255, 0.2)', 'rgba(255, 255, 255, 0.05)');

  return (
    <Flex position="relative" h="100vh" w="100vw" overflow="hidden">
      {/* Left Side: Form Container */}
      <Flex
        h="100%"
        w={{ base: '100%', md: '50vw', lg: '40vw' }}
        bg={formBg}
        zIndex={2}
        justifyContent="center"
        alignItems="center"
        direction="column"
        boxShadow="2xl"
        position="relative"
      >
        {children}
      </Flex>
      
      {/* Right Side: Rich Branding & Aesthetic Background */}
      <Flex
        display={{ base: 'none', md: 'flex' }}
        h="100%"
        w={{ md: '50vw', lg: '60vw' }}
        position="absolute"
        right="0px"
        bg={bgGradient}
        overflow="hidden"
        direction="column"
        justify="center"
        align="center"
        px="40px"
      >
        {/* Background Patterns (Dots) */}
        <Box
          position="absolute"
          top="0"
          left="0"
          right="0"
          bottom="0"
          opacity="0.1"
          backgroundImage="radial-gradient(circle at 2px 2px, white 1px, transparent 0)"
          backgroundSize="32px 32px"
        />

        {/* Floating Ambient Shapes */}
        <Box
          position="absolute"
          top="-10%"
          left="-10%"
          w="500px"
          h="500px"
          bg="whiteAlpha.200"
          borderRadius="full"
          filter="blur(60px)"
          animation="float 6s ease-in-out infinite"
        />
        <Box
          position="absolute"
          bottom="-20%"
          right="-10%"
          w="600px"
          h="600px"
          bg="brand.400"
          opacity="0.3"
          borderRadius="full"
          filter="blur(80px)"
          animation="float 8s ease-in-out infinite reverse"
        />

        {/* Content Container */}
        <VStack spacing={8} zIndex={3} maxW="600px" textAlign="center">
          <Box>
            <Heading fontSize={{ md: '3xl', lg: '5xl' }} color={textColor} fontWeight="bold" mb={4} lineHeight="1.2">
              {t('hero.welcome')} <Text as="span" color="orange.300">{t('hero.title')}</Text>
            </Heading>
            <Text fontSize="lg" color="whiteAlpha.800" maxW="450px" mx="auto">
              {t('hero.subtitle')}
            </Text>
          </Box>

          {/* Glassmorphism Feature Cards */}
          <Flex gap={6} mt={8} flexWrap="wrap" justify="center">
            {[
              { icon: MdOutlineDashboard, title: t('hero.features.modern.title'), desc: t('hero.features.modern.desc') },
              { icon: MdOutlineSecurity, title: t('hero.features.secure.title'), desc: t('hero.features.secure.desc') },
              { icon: MdOutlineSpeed, title: t('hero.features.fast.title'), desc: t('hero.features.fast.desc') },
            ].map((feature, idx) => (
              <Flex
                key={idx}
                direction="column"
                align="center"
                bg={glassBg}
                backdropFilter="blur(10px)"
                border="1px solid"
                borderColor={glassBorder}
                p={6}
                borderRadius="2xl"
                w="160px"
                transition="all 0.3s"
                _hover={{ transform: 'translateY(-5px)', bg: 'whiteAlpha.200' }}
              >
                <Flex bg="whiteAlpha.200" p={3} borderRadius="full" mb={4}>
                  <Icon as={feature.icon} w={6} h={6} color="white" />
                </Flex>
                <Text color="white" fontWeight="bold" mb={2}>{feature.title}</Text>
                <Text color="whiteAlpha.700" fontSize="sm">{feature.desc}</Text>
              </Flex>
            ))}
          </Flex>
        </VStack>
      </Flex>

      {/* Floating Controls */}
      <Flex position="fixed" top="30px" right="35px" gap="10px" zIndex="99">
        <LanguageSwitcher />
        <FixedPlugin bottom="auto" top="auto" />
      </Flex>
      
      {/* Animations */}
      <style>
        {`
        @keyframes float {
          0% { transform: translateY(0px) scale(1); }
          50% { transform: translateY(-20px) scale(1.05); }
          100% { transform: translateY(0px) scale(1); }
        }
        `}
      </style>
    </Flex>
  );
}

export default AuthIllustration;
