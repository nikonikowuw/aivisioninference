import { Box, Flex, useColorModeValue } from '@chakra-ui/react';
import React, { ReactNode } from 'react';
import LanguageSwitcher from 'components/i18n/LanguageSwitcher';
import FixedPlugin from 'components/fixedPlugin/FixedPlugin';

function CenteredAuth(props: { children: ReactNode; backgroundImage?: string }) {
  const { children, backgroundImage } = props;
  const authBg = useColorModeValue('gray.50', 'navy.900');
  
  return (
    <Flex
      minH="100vh"
      w="100%"
      bg={authBg}
      position="relative"
      justifyContent="center"
      alignItems="center"
      overflow="hidden"
      bgImage={backgroundImage ? `url(${backgroundImage})` : 'none'}
      bgSize="cover"
      bgPosition="center"
    >
      {/* Dark overlay for better contrast if background image exists */}
      {backgroundImage && (
        <Box
          position="absolute"
          top="0"
          left="0"
          w="100%"
          h="100%"
          bg="blackAlpha.200"
          zIndex="0"
        />
      )}
      
      {/* Decorative background elements (glows) */}
      <Box
        position="absolute"
        top="-10%"
        left="-10%"
        w="40vw"
        h="40vw"
        bg="brand.500"
        filter="blur(150px)"
        opacity={backgroundImage ? "0.15" : "0.05"}
        borderRadius="full"
        zIndex="0"
      />
      <Box
        position="absolute"
        bottom="-10%"
        right="-10%"
        w="40vw"
        h="40vw"
        bg="brand.400"
        filter="blur(150px)"
        opacity="0.05"
        borderRadius="full"
        zIndex="0"
      />
      
      <Box zIndex="1" w="100%" maxW="450px" px="20px">
        {children}
      </Box>

      <Flex position="fixed" top="30px" right="35px" gap="10px" zIndex="99">
        <LanguageSwitcher />
        <FixedPlugin bottom="auto" top="auto" />
      </Flex>
    </Flex>
  );
}

export default CenteredAuth;
