import React, { useState, type FormEvent } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Box,
  Button,
  Checkbox,
  Flex,
  FormControl,
  FormLabel,
  Icon,
  Input,
  InputGroup,
  InputRightElement,
  Text,
  useColorModeValue,
  useToast,
  Link,
  Image,
  VStack,
} from '@chakra-ui/react';
import Card from 'components/card/Card';
import CenteredAuth from 'layouts/auth/Centered';
import { MdOutlineRemoveRedEye } from 'react-icons/md';
import { RiEyeCloseLine } from 'react-icons/ri';
import { useAuth } from 'contexts/AuthContext';
import { NavLink } from 'react-router-dom';

import loginBg from 'assets/img/auth/login-bg.jpg';

function SignIn() {
  const { t } = useTranslation('auth');
  const textColor = useColorModeValue('navy.700', 'white');
  const textColorSecondary = useColorModeValue('gray.600', 'gray.400');
  const textColorBrand = useColorModeValue('brand.500', 'white');
  const brandStars = useColorModeValue('brand.500', 'brand.400');
  const cardBg = useColorModeValue('whiteAlpha.200', 'whiteAlpha.100');
  const cardBorder = useColorModeValue('whiteAlpha.300', 'whiteAlpha.200');

  const [show, setShow] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [rememberMe, setRememberMe] = useState(false);
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const toast = useToast();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await login(username, password, rememberMe);
      navigate('/admin/default');
    } catch (err) {
      toast({
        title: t('message.signInFailed'),
        description: err instanceof Error ? err.message : t('message.invalidCredentials'),
        status: 'error',
        duration: 3000,
        isClosable: true,
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <CenteredAuth backgroundImage={loginBg}>
      <Card
        p="40px"
        borderRadius="2xl"
        boxShadow="2xl"
        bg={cardBg}
        backdropFilter="blur(10px)"
        border="1px solid"
        borderColor={cardBorder}
      >
        <VStack spacing="30px" align="stretch">
          <Box textAlign="center">
            <Flex align="center" justify="center" mb="20px">
              <Image src="/favicon.ico" w="48px" h="48px" me="12px" />
              <Text
                fontSize="28px"
                fontWeight="800"
                color={textColor}
                letterSpacing="-0.5px"
              >
                Niko <Text as="span" color="brand.500">Admin</Text>
              </Text>
            </Flex>
            <Text
              color={textColorSecondary}
              fontWeight="400"
              fontSize="md"
            >
              {t('signIn.subtitle')}
            </Text>
          </Box>

          <FormControl as="form" onSubmit={handleSubmit}>
            <VStack spacing="20px" align="stretch">
              <Box>
                <FormLabel
                  display="flex"
                  ms="4px"
                  fontSize="sm"
                  fontWeight="600"
                  color={textColor}
                  mb="8px"
                >
                  {t('signIn.username.label')}<Text color={brandStars} ms="2px">*</Text>
                </FormLabel>
                <Input
                  isRequired
                  variant="auth"
                  fontSize="sm"
                  placeholder={t('signIn.username.placeholder')}
                  fontWeight="500"
                  size="lg"
                  h="50px"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                />
              </Box>

              <Box>
                <FormLabel
                  ms="4px"
                  fontSize="sm"
                  fontWeight="600"
                  color={textColor}
                  display="flex"
                  mb="8px"
                >
                  {t('signIn.password.label')}<Text color={brandStars} ms="2px">*</Text>
                </FormLabel>
                <InputGroup size="md">
                  <Input
                    isRequired
                    fontSize="sm"
                    placeholder={t('signIn.password.placeholder')}
                    size="lg"
                    h="50px"
                    type={show ? 'text' : 'password'}
                    variant="auth"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                  <InputRightElement display="flex" alignItems="center" h="50px">
                    <Icon
                      color={textColorSecondary}
                      _hover={{ cursor: 'pointer' }}
                      as={show ? RiEyeCloseLine : MdOutlineRemoveRedEye}
                      onClick={() => setShow(!show)}
                    />
                  </InputRightElement>
                </InputGroup>
              </Box>

              <Flex justifyContent="space-between" align="center">
                <Flex align="center">
                  <Checkbox
                    id="remember-login"
                    colorScheme="brand"
                    me="10px"
                    isChecked={rememberMe}
                    onChange={(e) => setRememberMe(e.target.checked)}
                  />
                  <Text
                    as="label"
                    htmlFor="remember-login"
                    fontWeight="500"
                    color={textColor}
                    fontSize="sm"
                    cursor="pointer"
                  >
                    {t('signIn.rememberMe')}
                  </Text>
                </Flex>
                <Link as={NavLink} to="/auth/forgot-password">
                  <Text
                    color={textColorBrand}
                    fontSize="sm"
                    fontWeight="600"
                    _hover={{ textDecoration: 'underline' }}
                    whiteSpace="nowrap"
                  >
                    {t('signIn.forgotPassword')}
                  </Text>
                </Link>
              </Flex>

              <Button
                type="submit"
                fontSize="md"
                variant="brand"
                fontWeight="600"
                w="100%"
                h="50px"
                isLoading={loading}
                borderRadius="16px"
                _hover={{
                  transform: 'translateY(-2px)',
                  boxShadow: 'lg',
                }}
                _active={{
                  transform: 'translateY(0)',
                }}
                transition="all 0.2s"
              >
                {t('signIn.submit')}
              </Button>
            </VStack>
          </FormControl>
          
          <Box textAlign="center">
             <Text color={textColorSecondary} fontSize="sm">
               © 2026 Niko Admin. All rights reserved.
             </Text>
          </Box>
        </VStack>
      </Card>
    </CenteredAuth>
  );
}

export default SignIn;
