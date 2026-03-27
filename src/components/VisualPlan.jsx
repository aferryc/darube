import React from 'react';
import { analyzePlan } from '../utils/planParser';

const formatNumber = (value) => {
  if (value == null || !Number.isFinite(value)) return '-';
  return new Intl.NumberFormat().format(Math.round(value));
};

const formatMs = (value) => {
  if (value == null || !Number.isFinite(value)) return '-';
  const rounded = value >= 100 ? value.toFixed(0) : value >= 10 ? value.toFixed(1) : value.toFixed(2);
  return `${rounded} ms`;
};

const NodeCard = ({ data, isHot }) => {
  const details = [];
  if (data.accessType) details.push(`Access: ${data.accessType}`);
  if (data.key) details.push(`Key: ${data.key}`);
  if (data.possibleKeys) details.push(`Possible: ${data.possibleKeys}`);
  if (data.indexCond) details.push(`Index Cond: ${data.indexCond}`);
  if (data.filter) details.push(`Filter: ${data.filter}`);
  if (data.joinFilter) details.push(`Join Filter: ${data.joinFilter}`);
  if (data.hashCond) details.push(`Hash Cond: ${data.hashCond}`);
  if (data.mergeCond) details.push(`Merge Cond: ${data.mergeCond}`);
  if (data.attachedCondition) details.push(`Condition: ${data.attachedCondition}`);

  return (
    <div className={`plan-node${isHot ? ' hot' : ''}`}>
      <div className="plan-node-header">
        <span className="node-type">{data.type}</span>
        {isHot && <span className="node-hot">Hot</span>}
      </div>
      {(data.relation || data.alias) && (
        <div className="node-relation">
          on <strong>{data.relation}</strong> {data.alias && `(${data.alias})`}
        </div>
      )}
      <div className="node-metrics">
        <span>Rows: {formatNumber(data.rows)}</span>
        <span>Cost: {data.cost != null ? formatNumber(data.cost) : '-'}</span>
        {data.timeMs != null && <span>Time: {formatMs(data.timeMs)}</span>}
      </div>
      {details.length > 0 && (
        <div className="node-details">
          {details.slice(0, 3).map((detail, i) => (
            <div key={i} className="node-detail">{detail}</div>
          ))}
        </div>
      )}
    </div>
  );
};

const PlanTree = ({ node, hotIds }) => {
  if (!node) return null;

  return (
    <div className="plan-tree-container">
      <NodeCard data={node} isHot={hotIds?.has(node.id)} />
      {node.children && node.children.length > 0 && (
        <div className="plan-tree-children">
          {node.children.map((child, i) => (
            <div key={i} className="plan-tree-child">
              <PlanTree node={child} hotIds={hotIds} />
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

  const analysis = analyzePlan(plan);
  if (!analysis || analysis.raw || !analysis.root) {
    return (
      <div className="plan-raw-view">
        <h3 className="plan-raw-view-header">Raw Execution Plan</h3>
        {JSON.stringify(plan, null, 2)}
      </div>
    );
  }

  const { root, summary, hotspots, hotIds, metric, suggestions, dbType } = analysis;
  const totalLabel = metric === 'time'
    ? `Total Time: ${formatMs(summary?.totalTimeMs)}`
    : metric === 'cost'
      ? `Total Cost: ${formatNumber(summary?.totalCost)}`
      : `Total Rows: ${formatNumber(summary?.totalRows)}`;

  return (
    <div className="visual-plan-canvas">
      <div className="plan-summary">
        <div className="plan-summary-row">
          <div className="plan-summary-title">Execution Summary</div>
          <div className="plan-summary-meta">{dbType === 'postgres' ? 'Postgres' : 'MySQL/MariaDB'}</div>
        </div>
        <div className="plan-summary-metrics">
          <span>{totalLabel}</span>
          {summary?.totalRows != null && metric !== 'rows' && (
            <span>Rows: {formatNumber(summary.totalRows)}</span>
          )}
        </div>
        {hotspots?.length > 0 && (
          <div className="plan-hotspots">
            <div className="plan-hotspots-title">Hotspots</div>
            <div className="plan-hotspots-list">
              {hotspots.map((hotspot) => (
                <div key={hotspot.id} className="plan-hotspot-item">
                  <span className="plan-hotspot-name">
                    {hotspot.type}{hotspot.relation ? ` on ${hotspot.relation}` : ''}
                  </span>
                  <span className="plan-hotspot-metric">
                    {hotspot.metricType === 'time'
                      ? formatMs(hotspot.metric)
                      : formatNumber(hotspot.metric)}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}
        {suggestions?.length > 0 && (
          <div className="plan-suggestions">
            <div className="plan-suggestions-title">Index Suggestions</div>
            <div className="plan-suggestions-list">
              {suggestions.slice(0, 3).map((item, i) => (
                <div key={i} className="plan-suggestion-item">{item}</div>
              ))}
            </div>
          </div>
        )}
      </div>
      <div className="tree-scroll-wrapper">
        <PlanTree node={root} hotIds={hotIds} />
      </div>
    </div>
  );
};
