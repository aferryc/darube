import React from 'react';
import { parsePostgresNode } from '../utils/planParser';

const NodeCard = ({ data }) => {
  return (
    <div className="plan-node">
      <div className="plan-node-header">
        <span className="node-type">{data.type}</span>
        {data.time && <span className="node-time">{data.time}</span>}
      </div>
      {(data.relation || data.alias) && (
        <div className="node-relation">
          on <strong>{data.relation}</strong> {data.alias && `(${data.alias})`}
        </div>
      )}
      <div className="node-metrics">
        <span>Rows: {data.rows}</span>
        <span>Cost: {data.cost}</span>
      </div>
    </div>
  );
};

const PlanTree = ({ node }) => {
  if (!node) return null;

  return (
    <div className="plan-tree-container">
      <NodeCard data={node} />
      {node.children && node.children.length > 0 && (
        <div className="plan-tree-children">
          {node.children.map((child, i) => (
            <div key={i} className="plan-tree-child">
              <PlanTree node={child} />
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

// Generic visual plan component
export const VisualPlan = ({ plan }) => {
  if (!plan) return <div className="plan-summary">No plan available.</div>;

  let rootNode = null;
  let isPostgres = false;

  // Try Postgres formatting
  const pgRoot = Array.isArray(plan) ? plan[0].Plan : plan.Plan;
  if (pgRoot && pgRoot["Node Type"]) {
    isPostgres = true;
    rootNode = parsePostgresNode(pgRoot);
  }

  // Very very messy fallback if it's MySQL or unsupported
  if (!rootNode) {
    return (
      <div className="plan-raw-view">
        <h3 className="plan-raw-view-header">Raw Execution Plan</h3>
        {JSON.stringify(plan, null, 2)}
      </div>
    );
  }

  return (
    <div className="visual-plan-canvas">
      {isPostgres && (
        <div className="plan-summary">
           Total Cost: {rootNode.cost} | Plan Rows: {rootNode.rows} {rootNode.time && `| Time: ${rootNode.time}`}
        </div>
      )}
      <div className="tree-scroll-wrapper">
        <PlanTree node={rootNode} />
      </div>
    </div>
  );
};
