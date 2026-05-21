import {
  Badge,
  Box,
  Button,
  Checkbox,
  Flex,
  HStack,
  Icon,
  IconButton,
  Text,
  VStack,
  useColorModeValue,
  Collapse,
  useDisclosure,
  Wrap,
  WrapItem,
  Tooltip,
  Divider,
} from '@chakra-ui/react';
import { ChevronDownIcon, ChevronRightIcon } from '@chakra-ui/icons';
import { MdMenu, MdRadioButtonChecked, MdExpandMore, MdExpandLess, MdCheckCircle, MdRemoveCircleOutline } from 'react-icons/md';
import { useState, useMemo, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { type Permission } from 'services/api';
import Card from 'components/card/Card';

interface PermissionTreeProps {
  tree: Permission[];
  selectedIds: string[];
  onChange: (selectedIds: string[]) => void;
}

/**
 * 递归获取所有后代节点的 ID
 */
function getAllDescendantIds(node: Permission): string[] {
  const ids: string[] = [];
  const walk = (children: Permission[]) => {
    for (const child of children) {
      ids.push(child.id);
      if (child.children) walk(child.children);
    }
  };
  if (node.children) walk(node.children);
  return ids;
}

/**
 * 权限树节点组件
 */
const PermissionNode = ({
  node,
  depth,
  selectedIds,
  onToggle,
  expandedIds,
  onToggleExpand,
}: {
  node: Permission;
  depth: number;
  selectedIds: string[];
  onToggle: (id: string) => void;
  expandedIds: Set<string>;
  onToggleExpand: (id: string) => void;
}) => {
  const { t: tMenu } = useTranslation('menu');
  const isExpanded = expandedIds.has(node.id);
  
  const isLeaf = !node.children || node.children.length === 0;
  const descendantIds = useMemo(() => getAllDescendantIds(node), [node]);
  
  const isChecked = selectedIds.includes(node.id);
  const checkedDescendants = descendantIds.filter((id) => selectedIds.includes(id));
  
  const allChecked = isLeaf ? isChecked : (isChecked && checkedDescendants.length === descendantIds.length);
  const someChecked = isChecked || checkedDescendants.length > 0;
  const isIndeterminate = someChecked && !allChecked;

  const bgHover = useColorModeValue('gray.50', 'whiteAlpha.50');
  const lineConnectorColor = useColorModeValue('gray.200', 'gray.600');
  const menuIconColor = useColorModeValue('brand.500', 'brand.300');
  const buttonIconColor = useColorModeValue('orange.500', 'orange.300');
  const textColor = useColorModeValue('navy.700', 'white');

  const getDisplayName = (p: Permission) => {
    if (p.type === 'menu') {
      return tMenu(p.code, { defaultValue: p.name });
    }
    return p.name;
  };

  // 分离孩子：菜单和按钮
  const { menuChildren, buttonChildren } = useMemo(() => {
    const menus = (node.children || []).filter(c => c.type === 'menu');
    const buttons = (node.children || []).filter(c => c.type !== 'menu');
    return { menuChildren: menus, buttonChildren: buttons };
  }, [node.children]);

  const isTopLevel = depth === 0;

  return (
    <Box w="100%" position="relative">
      <Flex 
        align="center" 
        py={2} 
        px={2} 
        borderRadius="lg" 
        _hover={{ bg: bgHover }}
        transition="all 0.2s"
        cursor="pointer"
        onClick={() => onToggleExpand(node.id)}
      >
        <HStack spacing={2} flex="1">
          {!isLeaf ? (
            <Icon
              as={isExpanded ? ChevronDownIcon : ChevronRightIcon}
              w="18px"
              h="18px"
              color="gray.400"
            />
          ) : (
            <Box w="18px" />
          )}
          
          <Checkbox
            isChecked={allChecked}
            isIndeterminate={isIndeterminate}
            onChange={(e) => {
              e.stopPropagation();
              onToggle(node.id);
            }}
            colorScheme="brand"
            onClick={(e) => e.stopPropagation()}
          />

          <Icon
            as={node.type === 'menu' ? MdMenu : MdRadioButtonChecked}
            color={node.type === 'menu' ? menuIconColor : buttonIconColor}
            w="16px"
            h="16px"
          />
          
          <Tooltip label={node.code} placement="top" hasArrow>
            <Text
              fontSize="sm"
              fontWeight={isTopLevel ? 'bold' : 'medium'}
              color={textColor}
              noOfLines={1}
            >
              {getDisplayName(node)}
            </Text>
          </Tooltip>
          
          {!isTopLevel && (
             <Text fontSize="xs" color="gray.400" fontWeight="normal">
               {node.code}
             </Text>
          )}
        </HStack>
      </Flex>

      {!isLeaf && (
        <Collapse in={isExpanded} animateOpacity>
          <Box 
            position="relative" 
            ml={4} 
            pl={4} 
            mt={1}
            borderLeft="1px solid"
            borderColor={lineConnectorColor}
          >
            <VStack align="stretch" spacing={0}>
              {/* 渲染菜单类的子节点 */}
              {menuChildren.map((child) => (
                <PermissionNode
                  key={child.id}
                  node={child}
                  depth={depth + 1}
                  selectedIds={selectedIds}
                  onToggle={onToggle}
                  expandedIds={expandedIds}
                  onToggleExpand={onToggleExpand}
                />
              ))}
              
              {/* 如果有按钮类的子节点，横向排列 */}
              {buttonChildren.length > 0 && (
                <Box py={3} px={2} mt={1} borderRadius="md" bg={useColorModeValue('gray.50', 'whiteAlpha.50')}>
                  <Wrap spacing={4}>
                    {buttonChildren.map((btn) => (
                      <WrapItem key={btn.id}>
                        <HStack 
                          spacing={2} 
                          px={2} 
                          py={1} 
                          borderRadius="md" 
                          _hover={{ bg: useColorModeValue('white', 'whiteAlpha.100') }}
                          transition="all 0.2s"
                        >
                          <Checkbox
                            isChecked={selectedIds.includes(btn.id)}
                            onChange={() => onToggle(btn.id)}
                            size="sm"
                            colorScheme="orange"
                          />
                          <Tooltip label={btn.code} hasArrow>
                            <Text fontSize="xs" color={useColorModeValue('gray.600', 'gray.400')} fontWeight="medium">
                              {getDisplayName(btn)}
                            </Text>
                          </Tooltip>
                        </HStack>
                      </WrapItem>
                    ))}
                  </Wrap>
                </Box>
              )}
            </VStack>
          </Box>
        </Collapse>
      )}
    </Box>
  );
};

export default function PermissionTree({ tree, selectedIds, onChange }: PermissionTreeProps) {
  const { t } = useTranslation('common');
  const { t: tMenu } = useTranslation('menu');
  const borderColor = useColorModeValue('gray.200', 'whiteAlpha.100');
  const headerBg = useColorModeValue('gray.50', 'whiteAlpha.100');
  
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  // 默认展开第一层
  useEffect(() => {
    if (tree.length > 0 && expandedIds.size === 0) {
      setExpandedIds(new Set(tree.map(n => n.id)));
    }
  }, [tree]);

  const handleToggleExpand = (id: string) => {
    const newExpanded = new Set(expandedIds);
    if (newExpanded.has(id)) {
      newExpanded.delete(id);
    } else {
      newExpanded.add(id);
    }
    setExpandedIds(newExpanded);
  };

  const handleToggle = (id: string) => {
    const findNode = (nodes: Permission[], targetId: string): Permission | null => {
      for (const n of nodes) {
        if (n.id === targetId) return n;
        if (n.children) {
          const found = findNode(n.children, targetId);
          if (found) return found;
        }
      }
      return null;
    };

    const targetNode = findNode(tree, id);
    if (!targetNode) return;

    const descendantIds = getAllDescendantIds(targetNode);
    const allAffectedIds = [id, ...descendantIds];
    const isCurrentlyChecked = selectedIds.includes(id);

    if (isCurrentlyChecked) {
      onChange(selectedIds.filter((sid) => !allAffectedIds.includes(sid)));
    } else {
      const newSet = new Set([...selectedIds, ...allAffectedIds]);
      const addParents = (nodes: Permission[], targetId: string, path: string[]): boolean => {
        for (const n of nodes) {
          if (n.id === targetId) {
            path.forEach(pid => newSet.add(pid));
            return true;
          }
          if (n.children && addParents(n.children, targetId, [...path, n.id])) return true;
        }
        return false;
      };
      addParents(tree, id, []);
      onChange(Array.from(newSet));
    }
  };

  const expandAll = () => {
    const allIds = new Set<string>();
    const walk = (nodes: Permission[]) => {
      for (const n of nodes) {
        if (n.children && n.children.length > 0) {
          allIds.add(n.id);
          walk(n.children);
        }
      }
    };
    walk(tree);
    setExpandedIds(allIds);
  };

  const collapseAll = () => {
    setExpandedIds(new Set());
  };

  const selectAll = () => {
    const allIds: string[] = [];
    const walk = (nodes: Permission[]) => {
      for (const n of nodes) {
        allIds.push(n.id);
        if (n.children) walk(n.children);
      }
    };
    walk(tree);
    onChange(allIds);
  };

  const deselectAll = () => {
    onChange([]);
  };

  return (
    <VStack spacing={6} align="stretch" w="100%">
      <Flex justify="space-between" align="center" px={2} py={2} bg={headerBg} borderRadius="xl">
        <HStack spacing={4}>
          <Button 
            size="sm" 
            variant="ghost" 
            leftIcon={<MdExpandMore />} 
            onClick={expandAll}
            borderRadius="lg"
          >
            {t('button.expandAll', { defaultValue: '全部展开' })}
          </Button>
          <Button 
            size="sm" 
            variant="ghost" 
            leftIcon={<MdExpandLess />} 
            onClick={collapseAll}
            borderRadius="lg"
          >
            {t('button.collapseAll', { defaultValue: '全部折叠' })}
          </Button>
        </HStack>
        <HStack spacing={2}>
          <Button 
            size="sm" 
            variant="outline" 
            colorScheme="brand" 
            leftIcon={<MdCheckCircle />} 
            onClick={selectAll}
            borderRadius="lg"
          >
            {t('permissions.selectAll')}
          </Button>
          <Button 
            size="sm" 
            variant="outline" 
            colorScheme="gray" 
            leftIcon={<MdRemoveCircleOutline />} 
            onClick={deselectAll}
            borderRadius="lg"
          >
            {t('permissions.deselectAll')}
          </Button>
        </HStack>
      </Flex>

      {tree.map((topNode) => (
        <Card
          key={topNode.id}
          p={0}
          overflow="hidden"
          variant="outline"
          boxShadow="sm"
          _hover={{ 
            boxShadow: 'md',
            borderColor: 'brand.300'
          }}
          transition="all 0.3s"
        >
          <Box 
            px={5} 
            py={4} 
            bg={headerBg}
            borderBottom="1px solid"
            borderColor={borderColor}
          >
            <HStack justify="space-between">
              <HStack spacing={3}>
                <Checkbox
                  isChecked={selectedIds.includes(topNode.id) && getAllDescendantIds(topNode).every(id => selectedIds.includes(id))}
                  isIndeterminate={selectedIds.includes(topNode.id) || getAllDescendantIds(topNode).some(id => selectedIds.includes(id))}
                  onChange={() => handleToggle(topNode.id)}
                  colorScheme="brand"
                />
                <Icon as={MdMenu} color="brand.500" w="22px" h="22px" />
                <Text fontWeight="bold" fontSize="lg" color={useColorModeValue('navy.700', 'white')}>
                  {topNode.type === 'menu' ? tMenu(topNode.code, { defaultValue: topNode.name }) : topNode.name}
                </Text>
              </HStack>
              <Badge 
                variant="subtle" 
                colorScheme="brand" 
                borderRadius="full" 
                px={3} 
                py={1}
                fontSize="xs"
              >
                {topNode.code}
              </Badge>
            </HStack>
          </Box>
          
          <Box p={4}>
            <PermissionNode
              node={topNode}
              depth={0}
              selectedIds={selectedIds}
              onToggle={handleToggle}
              expandedIds={expandedIds}
              onToggleExpand={handleToggleExpand}
            />
          </Box>
        </Card>
      ))}
    </VStack>
  );
}

