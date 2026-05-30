import {
  Badge,
  Box,
  Button,
  Divider,
  FormControl,
  FormLabel,
  HStack,
  Input,
  Select,
  SimpleGrid,
  Switch,
  Text,
  useColorModeValue,
  useToast,
} from '@chakra-ui/react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { mailConfigApi, type MailConfig } from 'services/api';

type FormState = Partial<MailConfig> & {
  smtp_password?: string;
  imap_password?: string;
};

export default function MailConfigPage() {
  const { t } = useTranslation('modules/mail-config');
  const toast = useToast();
  const textColor = useColorModeValue('navy.700', 'white');
  const bgCard = useColorModeValue('white', 'navy.800');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [testingSMTP, setTestingSMTP] = useState(false);
  const [testingIMAP, setTestingIMAP] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [testEmail, setTestEmail] = useState('');
  const [smtpPwdConfigured, setSmtpPwdConfigured] = useState(false);
  const [imapPwdConfigured, setImapPwdConfigured] = useState(false);
  const [form, setForm] = useState<FormState>({});

  useEffect(() => {
    loadConfig();
  }, []);

  async function loadConfig() {
    setLoading(true);
    try {
      const cfg = await mailConfigApi.get();
      setForm(cfg);
      setSmtpPwdConfigured(cfg.smtp_password_configured);
      setImapPwdConfigured(cfg.imap_password_configured);
      if (cfg.from_address) setTestEmail(cfg.from_address);
    } catch (err) {
      toast({ title: t('message.loadFailed'), description: toastErrorDesc(err), status: 'error' });
    } finally {
      setLoading(false);
    }
  }

  const setField = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm(prev => ({ ...prev, [key]: value }));
  };

  const toastErrorDesc = (err: unknown) => err instanceof Error ? err.message : '';

  async function save() {
    setSaving(true);
    try {
      const next = await mailConfigApi.save(form);
      setForm(next);
      setSmtpPwdConfigured(next.smtp_password_configured);
      setImapPwdConfigured(next.imap_password_configured);
      toast({ title: t('message.saved'), status: 'success' });
    } catch (err) {
      toast({ title: t('message.saveFailed'), description: toastErrorDesc(err), status: 'error' });
    } finally {
      setSaving(false);
    }
  }

  async function testSMTP() {
    setTestingSMTP(true);
    try {
      await mailConfigApi.testSMTP(testEmail);
      toast({ title: t('message.smtpTestOk'), status: 'success' });
    } catch (err) {
      toast({ title: t('message.smtpTestFailed'), description: toastErrorDesc(err), status: 'error' });
    } finally {
      setTestingSMTP(false);
    }
  }

  async function testIMAP() {
    setTestingIMAP(true);
    try {
      await mailConfigApi.testIMAP();
      toast({ title: t('message.imapTestOk'), status: 'success' });
    } catch (err) {
      toast({ title: t('message.imapTestFailed'), description: toastErrorDesc(err), status: 'error' });
    } finally {
      setTestingIMAP(false);
    }
  }

  async function syncIMAP() {
    setSyncing(true);
    try {
      const res = await mailConfigApi.syncIMAP();
      toast({ title: t('message.syncOk', { count: res.synced }), status: 'success' });
    } catch (err) {
      toast({ title: t('message.syncFailed'), description: toastErrorDesc(err), status: 'error' });
    } finally {
      setSyncing(false);
    }
  }

  return (
    <Box pt={{ base: '130px', md: '80px', xl: '80px' }}>
      <HStack justify="space-between" mb="20px">
        <Text fontSize="2xl" fontWeight="bold" color={textColor}>{t('title')}</Text>
        <Button variant="brand" onClick={save} isLoading={saving || loading}>{t('actions.save')}</Button>
      </HStack>
      <Box bg={bgCard} borderRadius="16px" border="1px solid" borderColor={borderColor} p={6}>
        <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
          <FormControl display="flex" alignItems="center">
            <FormLabel mb={0}>{t('fields.enabled')}</FormLabel>
            <Switch isChecked={!!form.enabled} onChange={e => setField('enabled', e.target.checked)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.fromName')}</FormLabel>
            <Input value={form.from_name || ''} onChange={e => setField('from_name', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.fromAddress')}</FormLabel>
            <Input value={form.from_address || ''} onChange={e => setField('from_address', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.replyTo')}</FormLabel>
            <Input value={form.reply_to || ''} onChange={e => setField('reply_to', e.target.value)} />
          </FormControl>
        </SimpleGrid>

        <Divider my={6} />
        <Text fontWeight="bold" mb={3}>{t('section.smtp')}</Text>
        <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
          <FormControl display="flex" alignItems="center">
            <FormLabel mb={0}>{t('fields.smtpEnabled')}</FormLabel>
            <Switch isChecked={!!form.smtp_enabled} onChange={e => setField('smtp_enabled', e.target.checked)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.smtpHost')}</FormLabel>
            <Input value={form.smtp_host || ''} onChange={e => setField('smtp_host', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.smtpPort')}</FormLabel>
            <Input
              type="number"
              value={Number.isFinite(form.smtp_port) ? form.smtp_port : ''}
              onChange={e => {
                const val = e.target.value;
                setField('smtp_port', val === '' ? undefined : Number(val));
              }}
            />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.smtpUsername')}</FormLabel>
            <Input value={form.smtp_username || ''} onChange={e => setField('smtp_username', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.smtpPassword')}</FormLabel>
            <Input type="password" value={form.smtp_password || ''} placeholder={smtpPwdConfigured ? t('message.passwordConfigured') : ''} onChange={e => setField('smtp_password', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.smtpEncryption')}</FormLabel>
            <Select value={form.smtp_encryption || 'starttls'} onChange={e => setField('smtp_encryption', e.target.value)}>
              <option value="none">{t('encryption.none')}</option>
              <option value="starttls">{t('encryption.starttls')}</option>
              <option value="tls">{t('encryption.tls')}</option>
            </Select>
          </FormControl>
        </SimpleGrid>

        <Divider my={6} />
        <Text fontWeight="bold" mb={3}>{t('section.imap')}</Text>
        <SimpleGrid columns={{ base: 1, md: 2 }} spacing={4}>
          <FormControl display="flex" alignItems="center">
            <FormLabel mb={0}>{t('fields.imapEnabled')}</FormLabel>
            <Switch isChecked={!!form.imap_enabled} onChange={e => setField('imap_enabled', e.target.checked)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapHost')}</FormLabel>
            <Input value={form.imap_host || ''} onChange={e => setField('imap_host', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapPort')}</FormLabel>
            <Input
              type="number"
              value={Number.isFinite(form.imap_port) ? form.imap_port : ''}
              onChange={e => {
                const val = e.target.value;
                setField('imap_port', val === '' ? undefined : Number(val));
              }}
            />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapUsername')}</FormLabel>
            <Input value={form.imap_username || ''} onChange={e => setField('imap_username', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapPassword')}</FormLabel>
            <Input type="password" value={form.imap_password || ''} placeholder={imapPwdConfigured ? t('message.passwordConfigured') : ''} onChange={e => setField('imap_password', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapEncryption')}</FormLabel>
            <Select value={form.imap_encryption || 'tls'} onChange={e => setField('imap_encryption', e.target.value)}>
              <option value="none">{t('encryption.none')}</option>
              <option value="starttls">{t('encryption.starttls')}</option>
              <option value="tls">{t('encryption.tls')}</option>
            </Select>
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapMailbox')}</FormLabel>
            <Input value={form.imap_mailbox || 'INBOX'} onChange={e => setField('imap_mailbox', e.target.value)} />
          </FormControl>
          <FormControl>
            <FormLabel>{t('fields.imapSyncMinutes')}</FormLabel>
            <Input type="number" value={form.imap_sync_minutes || 10} onChange={e => setField('imap_sync_minutes', Number(e.target.value))} />
          </FormControl>
        </SimpleGrid>

        <Divider my={6} />
        <HStack spacing={3} align="end">
          <FormControl maxW="360px">
            <FormLabel>{t('fields.testEmail')}</FormLabel>
            <Input value={testEmail} onChange={e => setTestEmail(e.target.value)} />
          </FormControl>
          <Button onClick={testSMTP} isLoading={testingSMTP}>{t('actions.testSMTP')}</Button>
          <Button onClick={testIMAP} isLoading={testingIMAP}>{t('actions.testIMAP')}</Button>
          <Button onClick={syncIMAP} isLoading={syncing}>{t('actions.syncIMAP')}</Button>
          <Badge>{t('message.secureHint')}</Badge>
        </HStack>
      </Box>
    </Box>
  );
}
