import { useState } from 'react';
import {
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalCloseButton,
  Flex,
  Box,
  VStack,
  Icon,
  Text,
  useColorModeValue,
  HStack,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import { FiUser, FiLock, FiChevronRight } from 'react-icons/fi';
import PersonalInfo from '../../views/admin/profile/components/PersonalInfo';
import SecuritySettings from '../../views/admin/profile/components/SecuritySettings';
import { type User } from 'services/api';

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  user: User;
}

type TabType = 'personal' | 'security';

export default function ProfileModal({ isOpen, onClose, user }: ProfileModalProps) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<TabType>('personal');

  const textColor = useColorModeValue('secondaryGray.900', 'white');
  const secondaryColor = useColorModeValue('gray.600', 'gray.400');
  const activeBg = useColorModeValue('white', 'navy.700');
  const modalBg = useColorModeValue('white', 'navy.800');
  const navBg = useColorModeValue('gray.50', 'navy.900');
  const activeShadow = useColorModeValue(
    '0px 18px 40px rgba(112, 144, 176, 0.12)',
    'none'
  );

  const navItems = [
    { id: 'personal', label: t('common:profile.basicInfo'), icon: FiUser },
    { id: 'security', label: t('common:profile.security'), icon: FiLock },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} size="4xl" isCentered motionPreset="slideInBottom">
      <ModalOverlay backdropFilter="blur(4px)" />
      <ModalContent borderRadius="2xl" overflow="hidden" bg={modalBg} minH="500px">
        <ModalHeader borderBottomWidth="1px" py={4}>
          <HStack spacing={2}>
            <Text fontSize="lg" fontWeight="bold" color={textColor}>
              {t('common:profile.title')}
            </Text>
            <Icon as={FiChevronRight} color="gray.400" />
            <Text fontSize="lg" fontWeight="bold" color="brand.500">
              {navItems.find((i) => i.id === activeTab)?.label}
            </Text>
          </HStack>
        </ModalHeader>
        <ModalCloseButton />

        <ModalBody p={0}>
          <Flex direction={{ base: 'column', md: 'row' }} minH="500px">
            {/* Left Sidebar Navigation */}
            <Box
              w={{ base: '100%', md: '240px' }}
              bg={navBg}
              p={4}
              borderRightWidth={{ base: 0, md: '1px' }}
              borderBottomWidth={{ base: '1px', md: 0 }}
            >
              <VStack spacing={2} align="stretch">
                {navItems.map((item) => (
                  <Box
                    key={item.id}
                    onClick={() => setActiveTab(item.id as TabType)}
                    px={4}
                    py={3}
                    borderRadius="xl"
                    cursor="pointer"
                    bg={activeTab === item.id ? activeBg : 'transparent'}
                    boxShadow={activeTab === item.id ? activeShadow : 'none'}
                    _hover={{
                      bg: activeTab === item.id ? activeBg : useColorModeValue('gray.200', 'whiteAlpha.100'),
                    }}
                    color={activeTab === item.id ? 'brand.500' : secondaryColor}
                    transition="all 0.2s"
                  >
                    <Flex align="center">
                      <Icon
                        as={item.icon}
                        boxSize={5}
                        mr={3}
                        color={activeTab === item.id ? 'brand.500' : 'gray.400'}
                      />
                      <Text fontWeight={activeTab === item.id ? '700' : '500'} fontSize="sm">
                        {item.label}
                      </Text>
                    </Flex>
                  </Box>
                ))}
              </VStack>
            </Box>

            {/* Right Content Area */}
            <Box flex={1} p={8} overflowY="auto" maxH="600px">
              {activeTab === 'personal' && <PersonalInfo />}
              {activeTab === 'security' && <SecuritySettings />}
            </Box>
          </Flex>
        </ModalBody>
      </ModalContent>
    </Modal>
  );
}
