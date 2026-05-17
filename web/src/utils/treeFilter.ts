import { Permission } from 'services/api';

export interface FilteredNode extends Permission {
  isAncestor?: boolean;
}

/**
 * nodeMatches 判断节点是否匹配筛选条件
 *
 * 【核心功能】对节点的 name、code、path、method 字段进行关键字 LIKE 匹配（不区分大小写），
 * 并支持 type 精确匹配。用于 Permission 树形搜索。
 *
 * @param node - 要检查的权限节点
 * @param keyword - 搜索关键字（可选）
 * @param type - 类型筛选（可选）
 * @returns 是否匹配
 */
export function nodeMatches(
  node: Permission,
  keyword?: string,
  type?: string,
): boolean {
  // 类型精确匹配
  if (type && node.type !== type) {
    return false;
  }

  // 关键字匹配
  if (keyword) {
    const lowerKeyword = keyword.toLowerCase();
    const fields = [node.name, node.code, node.path, node.method].filter(
      (f): f is string => f != null,
    );
    const matchesKeyword = fields.some(f => f.toLowerCase().includes(lowerKeyword));
    if (!matchesKeyword) {
      return false;
    }
  }

  return true;
}

/**
 * deepCopyChildren 深拷贝子树，避免修改原始 Permission 树
 *
 * 【核心功能】递归复制节点及其 children，确保返回的树与原始树完全独立。
 *
 * @param children - 原始子节点数组
 * @returns 深拷贝后的 FilteredNode 数组
 */
function deepCopyChildren(children: Permission[]): FilteredNode[] {
  return children.map(child => ({
    ...child,
    children: child.children ? deepCopyChildren(child.children) : child.children,
  }));
}

/**
 * filterTree 递归过滤权限树
 *
 * 【核心功能】树剪枝算法：递归过滤每个节点的 children，若自身匹配则保留完整子树；
 * 若自身不匹配但 children 过滤后非空则保留为祖先路径（低对比度）；否则剪除。
 * 自身匹配时不继续标记后代 isAncestor，因为此时用户需要看到该权限节点下的完整上下文。
 *
 * @param nodes - 权限节点数组
 * @param keyword - 搜索关键字（可选）
 * @param type - 类型筛选（可选）
 * @returns 过滤后的节点数组，标记了 isAncestor 属性
 */
export function filterTree(
  nodes: Permission[],
  keyword?: string,
  type?: string,
): FilteredNode[] {
  // 如果没有筛选条件，返回原始树的浅拷贝
  if (!keyword && !type) {
    return nodes.slice();
  }

  return nodes.reduce<FilteredNode[]>((acc, node) => {
    const selfMatch = nodeMatches(node, keyword, type);

    // 递归过滤 children
    const filteredChildren = node.children
      ? filterTree(node.children, keyword, type)
      : [];

    if (selfMatch) {
      // 自身匹配：保留全部 children，深拷贝避免修改原始树
      acc.push({
        ...node,
        children: node.children ? deepCopyChildren(node.children) : node.children,
        isAncestor: false,
      });
    } else if (filteredChildren.length > 0) {
      // 自身不匹配但有匹配的后代：保留为祖先路径
      acc.push({
        ...node,
        children: filteredChildren,
        isAncestor: true,
      });
    }
    // 否则：剪除整个子树

    return acc;
  }, []);
}
