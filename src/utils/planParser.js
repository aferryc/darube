const SQL_KEYWORDS = new Set([
  "AND", "OR", "NOT", "NULL", "TRUE", "FALSE", "LIKE", "ILIKE", "IN", "IS", "BETWEEN",
  "SIMILAR", "TO", "ANY", "ALL", "CASE", "WHEN", "THEN", "ELSE", "END", "AS",
  "ON", "JOIN", "INNER", "LEFT", "RIGHT", "FULL", "OUTER", "CROSS", "USING",
  "SELECT", "FROM", "WHERE", "GROUP", "BY", "ORDER", "LIMIT", "OFFSET",
]);

const normalizeNumber = (value) => {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return null;
};

/**
 * Recursively parses Postgres JSON EXPLAIN plan node.
 */
export function parsePostgresNode(node) {
  if (!node) return null;
  const loops = normalizeNumber(node["Actual Loops"]) || 1;
  const actualTime = normalizeNumber(node["Actual Total Time"]);
  const timeMs = actualTime != null ? actualTime * loops : null;
  return {
    type: node["Node Type"] || "Unknown Node",
    relation: node["Relation Name"] || "",
    alias: node["Alias"] || "",
    cost: normalizeNumber(node["Total Cost"]),
    rows: normalizeNumber(node["Actual Rows"]) ?? normalizeNumber(node["Plan Rows"]) ?? 0,
    timeMs,
    loops,
    width: normalizeNumber(node["Plan Width"]),
    filter: node["Filter"] || "",
    indexCond: node["Index Cond"] || "",
    joinFilter: node["Join Filter"] || "",
    hashCond: node["Hash Cond"] || "",
    mergeCond: node["Merge Cond"] || "",
    recheckCond: node["Recheck Cond"] || "",
    children: node.Plans ? node.Plans.map(parsePostgresNode) : [],
  };
}

const getAccessLabel = (accessType) => {
  switch ((accessType || "").toLowerCase()) {
    case "all":
      return "Seq Scan";
    case "index":
      return "Index Scan";
    case "range":
      return "Index Range";
    case "ref":
    case "eq_ref":
      return "Index Lookup";
    case "const":
      return "Const";
    case "system":
      return "System";
    default:
      return "Table Access";
  }
};

const parseMySQLTable = (table) => {
  if (!table) return null;
  return {
    type: getAccessLabel(table.access_type),
    relation: table.table_name || "",
    alias: table.table_alias || "",
    rows: normalizeNumber(table.rows_produced_per_join)
      ?? normalizeNumber(table.rows_examined_per_scan)
      ?? 0,
    cost: normalizeNumber(table.cost_info?.read_cost)
      ?? normalizeNumber(table.cost_info?.query_cost),
    timeMs: null,
    loops: 1,
    width: null,
    accessType: table.access_type || "",
    key: table.key || "",
    possibleKeys: Array.isArray(table.possible_keys) ? table.possible_keys.join(", ") : (table.possible_keys || ""),
    attachedCondition: table.attached_condition || "",
    children: [],
  };
};

const parseMySQLContainer = (label, container) => {
  if (!container) return null;
  const child = parseMySQLQueryBlock(container);
  if (!child) return null;
  return {
    type: label,
    relation: "",
    alias: "",
    rows: child.rows || 0,
    cost: child.cost,
    timeMs: null,
    loops: 1,
    width: null,
    children: [child],
  };
};

const parseMySQLNestedItem = (item) => {
  if (!item) return null;
  if (item.table) return parseMySQLTable(item.table);
  if (item.nested_loop) {
    const children = item.nested_loop.map(parseMySQLNestedItem).filter(Boolean);
    return {
      type: "Nested Loop",
      relation: "",
      alias: "",
      rows: children.reduce((sum, c) => sum + (c.rows || 0), 0),
      cost: children.reduce((sum, c) => sum + (c.cost || 0), 0) || null,
      timeMs: null,
      loops: 1,
      width: null,
      children,
    };
  }
  if (item.query_block) return parseMySQLQueryBlock(item.query_block);
  if (item.grouping_operation) return parseMySQLContainer("Grouping", item.grouping_operation);
  if (item.ordering_operation) return parseMySQLContainer("Ordering", item.ordering_operation);
  if (item.duplicates_removal) return parseMySQLContainer("Duplicates Removal", item.duplicates_removal);
  if (item.union_result) return parseMySQLContainer("Union", item.union_result);
  return null;
};

const parseMySQLQueryBlock = (block) => {
  if (!block) return null;

  if (block.table) return parseMySQLTable(block.table);
  if (block.nested_loop) {
    const children = block.nested_loop.map(parseMySQLNestedItem).filter(Boolean);
    return {
      type: "Query Block",
      relation: "",
      alias: "",
      rows: children.reduce((sum, c) => sum + (c.rows || 0), 0),
      cost: normalizeNumber(block.cost_info?.query_cost)
        ?? children.reduce((sum, c) => sum + (c.cost || 0), 0)
        ?? null,
      timeMs: null,
      loops: 1,
      width: null,
      children,
    };
  }
  if (block.grouping_operation) return parseMySQLContainer("Grouping", block.grouping_operation);
  if (block.ordering_operation) return parseMySQLContainer("Ordering", block.ordering_operation);
  if (block.duplicates_removal) return parseMySQLContainer("Duplicates Removal", block.duplicates_removal);
  if (block.union_result) return parseMySQLContainer("Union", block.union_result);
  return null;
};

export function parseMySQLPlan(plan) {
  if (!plan) return null;
  const root = plan.query_block ? parseMySQLQueryBlock(plan.query_block) : parseMySQLQueryBlock(plan);
  return root;
}

const collectNodes = (root) => {
  const nodes = [];
  const walk = (node) => {
    if (!node) return;
    nodes.push(node);
    if (node.children?.length) node.children.forEach(walk);
  };
  walk(root);
  return nodes;
};

const assignIds = (root) => {
  let idx = 0;
  const walk = (node) => {
    if (!node) return;
    node.id = `node-${idx++}`;
    if (node.children?.length) node.children.forEach(walk);
  };
  walk(root);
};

const extractColumns = (condition, relation, alias) => {
  if (!condition || typeof condition !== "string") return [];
  const cols = new Set();

  const qualifiedRe = /\"?([a-zA-Z_][\\w$]*)\"?\\.\"?([a-zA-Z_][\\w$]*)\"?/g;
  let match;
  while ((match = qualifiedRe.exec(condition)) !== null) {
    const table = match[1];
    const col = match[2];
    if (relation && table !== relation && alias && table !== alias) continue;
    cols.add(col);
  }

  if (cols.size === 0) {
    const tokenRe = /\"?([a-zA-Z_][\\w$]*)\"?/g;
    while ((match = tokenRe.exec(condition)) !== null) {
      const token = match[1];
      if (!token) continue;
      if (relation && token === relation) continue;
      if (alias && token === alias) continue;
      if (SQL_KEYWORDS.has(token.toUpperCase())) continue;
      cols.add(token);
    }
  }

  return Array.from(cols);
};

const collectPostgresSuggestions = (root) => {
  const suggestions = [];
  const seen = new Set();
  const nodes = collectNodes(root);
  nodes.forEach((node) => {
    if (!node?.type) return;
    const isSeq = node.type.toLowerCase().includes("seq scan")
      || node.type.toLowerCase().includes("bitmap heap scan");
    if (isSeq && node.filter && !node.indexCond) {
      const cols = extractColumns(node.filter, node.relation, node.alias);
      if (cols.length > 0) {
        const title = node.relation
          ? `Consider index on ${node.relation} (${cols.slice(0, 3).join(", ")})`
          : `Consider index on (${cols.slice(0, 3).join(", ")})`;
        if (!seen.has(title)) {
          seen.add(title);
          suggestions.push(title);
        }
      }
    }
    if (node.joinFilter) {
      const cols = extractColumns(node.joinFilter, node.relation, node.alias);
      if (cols.length > 0) {
        const title = `Join filter on (${cols.slice(0, 3).join(", ")}) may benefit from indexes`;
        if (!seen.has(title)) {
          seen.add(title);
          suggestions.push(title);
        }
      }
    }
  });
  return suggestions;
};

const collectMySQLSuggestions = (root) => {
  const suggestions = [];
  const seen = new Set();
  const nodes = collectNodes(root);
  nodes.forEach((node) => {
    if (!node?.type || !node.accessType) return;
    if (node.accessType.toLowerCase() === "all" && node.attachedCondition && !node.key) {
      const cols = extractColumns(node.attachedCondition, node.relation, node.alias);
      if (cols.length > 0) {
        const title = node.relation
          ? `Consider index on ${node.relation} (${cols.slice(0, 3).join(", ")})`
          : `Consider index on (${cols.slice(0, 3).join(", ")})`;
        if (!seen.has(title)) {
          seen.add(title);
          suggestions.push(title);
        }
      }
    }
  });
  return suggestions;
};

const computeSummary = (root) => {
  if (!root) return null;
  const totalTimeMs = normalizeNumber(root.timeMs);
  const totalCost = normalizeNumber(root.cost);
  const totalRows = normalizeNumber(root.rows);
  return {
    totalTimeMs,
    totalCost,
    totalRows,
  };
};

const computeHotspots = (root) => {
  if (!root) return { hotspots: [], hotIds: new Set(), metric: "rows" };
  const nodes = collectNodes(root);
  const summary = computeSummary(root);
  const hasTime = summary?.totalTimeMs != null;
  const hasCost = summary?.totalCost != null;
  const metric = hasTime ? "time" : hasCost ? "cost" : "rows";

  const getMetricValue = (node) => {
    if (!node) return 0;
    if (metric === "time") return node.timeMs || 0;
    if (metric === "cost") return node.cost || 0;
    return node.rows || 0;
  };

  const totalValue = metric === "time"
    ? (summary?.totalTimeMs || 0)
    : metric === "cost"
      ? (summary?.totalCost || 0)
      : (summary?.totalRows || 0);

  const threshold = metric === "time"
    ? Math.max(50, totalValue * 0.2)
    : totalValue * 0.2;

  const hotIds = new Set();
  nodes.forEach((node) => {
    if (getMetricValue(node) >= threshold && getMetricValue(node) > 0) {
      hotIds.add(node.id);
    }
  });

  const hotspots = nodes
    .map((node) => ({
      id: node.id,
      type: node.type,
      relation: node.relation,
      metric: getMetricValue(node),
      metricType: metric,
    }))
    .filter((h) => h.metric > 0)
    .sort((a, b) => b.metric - a.metric)
    .slice(0, 3);

  return { hotspots, hotIds, metric };
};

export function analyzePlan(plan) {
  if (!plan) return null;
  const pgRoot = Array.isArray(plan) ? plan[0]?.Plan : plan?.Plan;
  if (pgRoot && pgRoot["Node Type"]) {
    const root = parsePostgresNode(pgRoot);
    assignIds(root);
    const summary = computeSummary(root);
    const { hotspots, hotIds, metric } = computeHotspots(root);
    const suggestions = collectPostgresSuggestions(root);
    return {
      dbType: "postgres",
      root,
      summary,
      hotspots,
      hotIds,
      metric,
      suggestions,
    };
  }

  const myRoot = parseMySQLPlan(plan);
  if (myRoot) {
    assignIds(myRoot);
    const summary = computeSummary(myRoot);
    const { hotspots, hotIds, metric } = computeHotspots(myRoot);
    const suggestions = collectMySQLSuggestions(myRoot);
    return {
      dbType: "mysql",
      root: myRoot,
      summary,
      hotspots,
      hotIds,
      metric,
      suggestions,
    };
  }

  return { raw: true };
}
