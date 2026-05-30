import {
  Badge,
  Box,
  Button,
  Checkbox,
  Flex,
  HStack,
  Icon,
  Text,
  VStack,
  useColorModeValue,
  Collapse,
  Wrap,
  WrapItem,
  Tooltip,
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

function getAllDescendantIds(node: Permission): string[] {
  return getAllNodeIds(node.children ?? []);
}

function getAllNodeIds(nodes: Permission[]): string[] {
  return nodes.flatMap((node) => [node.id, ...getAllNodeIds(node.children ?? [])]);
}

function getExpandableNodeIds(nodes: Permission[]): string[] {
  return nodes.flatMap((node) => {
    if (!node.children?.length) return [];
    return [node.id, ...getExpandableNodeIds(node.children)];
  });
}

function findPermissionNode(nodes: Permission[], targetId: string): Permission | null {
  for (const node of nodes) {
    if (node.id === targetId) return node;
    const found = findPermissionNode(node.children ?? [], targetId);
    if (found) return found;
  }
  return null;
}

function addAncestorIds(nodes: Permission[], targetId: string, selectedIds: Set<string>, path: string[] = []): boolean {
  for (const node of nodes) {
    if (node.id === targetId) {
      path.forEach((id) => selectedIds.add(id));
      return true;
    }
    if (addAncestorIds(node.children ?? [], targetId, selectedIds, [...path, node.id])) return true;
  }
  return false;
}

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
  const { t: tPermission } = useTranslation('modules/permissions');
  const isExpanded = expandedIds.has(node.id);

  const isLeaf = !node.children?.length;
  const descendantIds = useMemo(() => getAllDescendantIds(node), [node]);

  const isChecked = selectedIds.includes(node.id);
  const checkedDescendantsCount = descendantIds.filter((id) => selectedIds.includes(id)).length;

  const allChecked = isChecked && (isLeaf || checkedDescendantsCount === descendantIds.length);
  const isIndeterminate = !allChecked && (isChecked || checkedDescendantsCount > 0);

  const bgHover = useColorModeValue('gray.50', 'whiteAlpha.50');
  const lineConnectorColor = useColorModeValue('gray.200', 'gray.600');
  const menuIconColor = useColorModeValue('brand.500', 'brand.300');
  const buttonIconColor = useColorModeValue('orange.500', 'orange.300');
  const textColor = useColorModeValue('navy.700', 'white');
  const secondaryTextColor = useColorModeValue('gray.600', 'gray.400');
  const buttonHoverBg = useColorModeValue('white', 'whiteAlpha.100');

const getDisplayName = (p: Permission, tMenu: any, tPermission: any) => {
  if (p.type === 'menu') return tMenu(p.code, { defaultValue: p.name });
  return tPermission(`codes.${p.code.replace(/:/g, '.')}`, { defaultValue: p.name });
};

  const { menuChildren, buttonChildren } = useMemo(() => {
    const menus: Permission[] = [];
    const buttons: Permission[] = [];
    for (const child of node.children ?? []) {
      if (child.type === 'menu') {
        menus.push(child);
      } else {
        buttons.push(child);
      }
    }
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
              {getDisplayName(node, tMenu, tPermission)}
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

              {/* 如果有按钮类的子节点,横向排列 */}
              {buttonChildren.length > 0 && (
                <Box py={3} px={2} mt={1} borderRadius="md" bg={bgHover}>
                  <Wrap spacing={4}>
                    {buttonChildren.map((btn) => (
                      <WrapItem key={btn.id}>
                        <HStack
                          spacing={2}
                          px={2}
                          py={1}
                          borderRadius="md"
                          _hover={{ bg: buttonHoverBg }}
                          transition="all 0.2s"
                        >
                          <Checkbox
                            isChecked={selectedIds.includes(btn.id)}
                            onChange={() => onToggle(btn.id)}
                            size="sm"
                            colorScheme="orange"
                          />
                          <Tooltip label={btn.code} hasArrow>
                            <Text fontSize="xs" color={secondaryTextColor} fontWeight="medium">
                              {getDisplayName(btn, tMenu, tPermission)}
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
  const topNodeTextColor = useColorModeValue('navy.700', 'white');

  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [hasInitialized, setHasInitialized] = useState(false);

  useEffect(() => {
    if (tree.length > 0 && !hasInitialized) {
      setExpandedIds(new Set(tree.map((node) => node.id)));
      setHasInitialized(true);
    }
  }, [tree, hasInitialized]);

  const handleToggleExpand = (id: string) => {
    setExpandedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const handleToggle = (id: string) => {
    const targetNode = findPermissionNode(tree, id);
    if (!targetNode) return;

    const affectedIds = [id, ...getAllDescendantIds(targetNode)];
    if (selectedIds.includes(id)) {
      onChange(selectedIds.filter((selectedId) => !affectedIds.includes(selectedId)));
      return;
    }

    const nextSelectedIds = new Set([...selectedIds, ...affectedIds]);
    addAncestorIds(tree, id, nextSelectedIds);
    onChange(Array.from(nextSelectedIds));
  };

  const expandAll = () => {
    const allIds = getExpandableNodeIds(tree);
    setExpandedIds(new Set(allIds));
  };

  const collapseAll = () => {
    setExpandedIds(new Set());
  };

  const selectAll = () => {
    onChange(getAllNodeIds(tree));
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
            {t('permissions.expandAll')}
          </Button>
          <Button
            size="sm"
            variant="ghost"
            leftIcon={<MdExpandLess />}
            onClick={collapseAll}
            borderRadius="lg"
          >
            {t('permissions.collapseAll')}
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

      {tree.map((topNode) => {
        const descendantIds = getAllDescendantIds(topNode);
        const isTopNodeChecked = selectedIds.includes(topNode.id);
        const hasSelectedDescendants = descendantIds.some((id) => selectedIds.includes(id));
        const allDescendantsSelected = descendantIds.every((id) => selectedIds.includes(id));
        const isTopNodeFullyChecked = isTopNodeChecked && allDescendantsSelected;
        const isTopNodeIndeterminate = !isTopNodeFullyChecked && (isTopNodeChecked || hasSelectedDescendants);
        const isTopNodeExpanded = expandedIds.has(topNode.id);
        const topNodeName = topNode.type === 'menu'
          ? tMenu(topNode.code, { defaultValue: topNode.name })
          : t(`codes.${topNode.code.replace(/:/g, '.')}`, { ns: 'modules/permissions', defaultValue: topNode.name });

        return (
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
              <HStack
                justify="space-between"
                cursor="pointer"
                onClick={() => handleToggleExpand(topNode.id)}
                _hover={{ opacity: 0.8 }}
              >
                <HStack spacing={3}>
                  <Checkbox
                    isChecked={isTopNodeFullyChecked}
                    isIndeterminate={isTopNodeIndeterminate}
                    onChange={(e) => {
                      e.stopPropagation();
                      handleToggle(topNode.id);
                    }}
                    colorScheme="brand"
                  />
                  <Icon as={isTopNodeExpanded ? MdExpandMore : MdExpandLess} color="brand.500" w="22px" h="22px" />
                  <Text fontWeight="bold" fontSize="lg" color={topNodeTextColor}>
                    {topNodeName}
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

            <Collapse in={isTopNodeExpanded} animateOpacity>
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
            </Collapse>
          </Card>
        );
      })}
    </VStack>
  );
}

