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
  Heading,
  Icon,
  Input,
  InputGroup,
  InputLeftElement,
  InputRightElement,
  Text,
  VStack,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { HSeparator } from 'components/separator/Separator';
import DefaultAuth from 'layouts/auth/Default';
import { MdOutlineRemoveRedEye, MdOutlinePersonOutline, MdLockOutline } from 'react-icons/md';
import { RiEyeCloseLine } from 'react-icons/ri';
import { useAuth } from 'contexts/AuthContext';

function SignIn() {
  const { t } = useTranslation('auth');
  const textColor = useColorModeValue('navy.700', 'white');
  const textColorSecondary = 'gray.400';
  const brandStars = useColorModeValue('brand.500', 'brand.400');
  
  // Card styles
  const cardBg = useColorModeValue('white', 'navy.800');
  const cardBorder = useColorModeValue('gray.100', 'navy.700');
  const cardShadow = useColorModeValue('0px 18px 40px rgba(112, 144, 176, 0.12)', 'none');

  const [show, setShow] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();
  const toast = useToast();

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      await login(username, password);
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
    <DefaultAuth>
      <Flex
        w="100%"
        maxW="450px"
        mx="auto"
        h="100%"
        alignItems="center"
        justifyContent="center"
        flexDirection="column"
        px={{ base: '25px', md: '0px' }}
      >
        <Box w="100%" mb="8" textAlign="center">
          <Heading color={textColor} fontSize="36px" mb="10px">
            {t('signIn.title')}
          </Heading>
          <Text color={textColorSecondary} fontWeight="400" fontSize="md">
            {t('signIn.subtitle')}
          </Text>
        </Box>

        <Flex
          zIndex="2"
          direction="column"
          w="100%"
          bg={cardBg}
          p={{ base: 6, md: 10 }}
          boxShadow={cardShadow}
          border="1px solid"
          borderColor={cardBorder}
          borderRadius="2xl"
          mx="auto"
        >
          <Flex align="center" mb="25px">
            <HSeparator />
            <Text color="gray.400" mx="14px" fontSize="sm" fontWeight="500">
              {t('signIn.divider')}
            </Text>
            <HSeparator />
          </Flex>

          <FormControl as="form" onSubmit={handleSubmit}>
            <VStack spacing={5}>
              <Box w="100%">
                <FormLabel ms="4px" fontSize="sm" fontWeight="500" color={textColor} mb="8px">
                  {t('signIn.username.label')} <Text as="span" color={brandStars}>*</Text>
                </FormLabel>
                <InputGroup size="lg">
                  <InputLeftElement>
                    <Icon as={MdOutlinePersonOutline} color={textColorSecondary} w={5} h={5} />
                  </InputLeftElement>
                  <Input
                    isRequired
                    variant="auth"
                    fontSize="sm"
                    placeholder={t('signIn.username.placeholder')}
                    fontWeight="500"
                    value={username}
                    onChange={(e) => setUsername(e.target.value)}
                    borderRadius="xl"
                  />
                </InputGroup>
              </Box>

              <Box w="100%">
                <FormLabel ms="4px" fontSize="sm" fontWeight="500" color={textColor} mb="8px">
                  {t('signIn.password.label')} <Text as="span" color={brandStars}>*</Text>
                </FormLabel>
                <InputGroup size="lg">
                  <InputLeftElement>
                    <Icon as={MdLockOutline} color={textColorSecondary} w={5} h={5} />
                  </InputLeftElement>
                  <Input
                    isRequired
                    fontSize="sm"
                    placeholder={t('signIn.password.placeholder')}
                    type={show ? 'text' : 'password'}
                    variant="auth"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    borderRadius="xl"
                  />
                  <InputRightElement>
                    <Icon
                      color={textColorSecondary}
                      _hover={{ cursor: 'pointer', color: 'brand.500' }}
                      as={show ? RiEyeCloseLine : MdOutlineRemoveRedEye}
                      onClick={() => setShow(!show)}
                      w={5}
                      h={5}
                      transition="all 0.2s"
                    />
                  </InputRightElement>
                </InputGroup>
              </Box>

              <Flex w="100%" justify="space-between" align="center" mt="-2">
                <Checkbox colorScheme="brand" size="md">
                  <Text fontSize="sm" color={textColorSecondary}>{t('signIn.rememberMe')}</Text>
                </Checkbox>
                <Text color="brand.500" fontSize="sm" fontWeight="500" cursor="pointer" _hover={{ textDecoration: 'underline' }}>
                  {t('signIn.forgotPassword')}
                </Text>
              </Flex>

              <Button
                type="submit"
                fontSize="md"
                variant="brand"
                fontWeight="bold"
                w="100%"
                h="50px"
                mt="4"
                borderRadius="xl"
                isLoading={loading}
                boxShadow={useColorModeValue('0px 10px 20px rgba(66, 42, 251, 0.3)', 'none')}
                _hover={{ transform: 'translateY(-2px)', boxShadow: useColorModeValue('0px 14px 24px rgba(66, 42, 251, 0.4)', '0px 10px 20px rgba(66, 42, 251, 0.2)') }}
                transition="all 0.3s"
              >
                {t('signIn.submit')}
              </Button>
            </VStack>
          </FormControl>
        </Flex>
      </Flex>
    </DefaultAuth>
  );
}

export default SignIn;
