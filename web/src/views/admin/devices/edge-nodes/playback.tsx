import { useNavigate, useParams } from 'react-router-dom';
import { Box, Text, Flex, Button, useColorModeValue } from '@chakra-ui/react';
import Card from 'components/card/Card';
import PlaybackView from 'components/terminal/PlaybackView';

export default function EdgeNodePlaybackPage() {
  const navigate = useNavigate();
  const { id: nodeId, sessionId } = useParams<{ id: string; sessionId: string }>();
  const textColor = useColorModeValue('navy.700', 'white');

  if (!sessionId) {
    return (
      <Card px="24px" py="24px">
        <Text color={textColor} mb={4}>无效的会话 ID</Text>
        <Button variant="outline" size="sm" onClick={() => navigate(-1)}>返回</Button>
      </Card>
    );
  }

  return (
    <Box pt={{ base: '60px', md: '90px' }}>
      <Card px="24px" py="24px">
        <Flex mb={2} align="center" gap={2}>
          <Text color={textColor} fontSize="lg" fontWeight="bold">
            终端回放
          </Text>
        </Flex>

        <PlaybackView sessionId={sessionId} nodeId={nodeId} />
      </Card>
    </Box>
  );
}
