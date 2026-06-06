import {
  Box,
  Flex,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Text,
  useColorModeValue,
} from '@chakra-ui/react';
import { useTranslation } from 'react-i18next';
import {
  MdAccessTime,
  MdInfo,
  MdMonitor,
  MdNotifications,
  MdStorage,
  MdWifi,
  MdDeviceHub,
} from 'react-icons/md';

import NetworkConfigTab from './tabs/NetworkConfigTab';
import StatusTab from './tabs/StatusTab';
import StorageConfigTab from './tabs/StorageConfigTab';
import SystemInfoTab from './tabs/SystemInfoTab';
import TimeConfigTab from './tabs/TimeConfigTab';
import WebhookConfigTab from './tabs/WebhookConfigTab';
import GB28181ConfigTab from './tabs/GB28181ConfigTab';

export default function SystemConfig() {
  const { t } = useTranslation('modules/system');
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <Flex direction="column" w="100%">
        <Text
          color={textColor}
          fontSize="2xl"
          fontWeight="700"
          mb="24px"
        >
          {t('system-config')}
        </Text>

        <Box
          bg={bgCard}
          borderRadius="16px"
          border="1px solid"
          borderColor={borderColor}
          p="24px"
        >
          <Tabs variant="enclosed" colorScheme="brand">
            <TabList>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdMonitor />
                  <Text>{t('status.title')}</Text>
                </Flex>
              </Tab>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdWifi />
                  <Text>{t('network.title')}</Text>
                </Flex>
              </Tab>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdAccessTime />
                  <Text>{t('time.title')}</Text>
                </Flex>
              </Tab>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdNotifications />
                  <Text>{t('webhook.title')}</Text>
                </Flex>
              </Tab>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdStorage />
                  <Text>{t('storage.title')}</Text>
                </Flex>
              </Tab>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdInfo />
                  <Text>{t('info.title', { defaultValue: '系统信息' })}</Text>
                </Flex>
              </Tab>
              <Tab>
                <Flex align="center" gap={2}>
                  <MdDeviceHub />
                  <Text>GB28181</Text>
                </Flex>
              </Tab>
            </TabList>

            <TabPanels>
              <TabPanel px={0}>
                <StatusTab />
              </TabPanel>
              <TabPanel px={0}>
                <NetworkConfigTab />
              </TabPanel>
              <TabPanel px={0}>
                <TimeConfigTab />
              </TabPanel>
              <TabPanel px={0}>
                <WebhookConfigTab />
              </TabPanel>
              <TabPanel px={0}>
                <StorageConfigTab />
              </TabPanel>
              <TabPanel px={0}>
                <SystemInfoTab />
              </TabPanel>
              <TabPanel px={0}>
                <GB28181ConfigTab />
              </TabPanel>
            </TabPanels>
          </Tabs>
        </Box>
      </Flex>
    </Box>
  );
}
