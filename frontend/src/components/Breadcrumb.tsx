import React from 'react';

interface BreadcrumbProps {
  path: string;
  onNavigate: (path: string) => void;
}

export const Breadcrumb: React.FC<BreadcrumbProps> = ({ path, onNavigate }) => {
  const parts = path.split('/').filter(Boolean);
  
  return (
    <div className="breadcrumb">
      <span onClick={() => onNavigate('/')} style={{ cursor: 'pointer' }}>/</span>
      {parts.map((part, i) => (
        <React.Fragment key={i}>
          <span className="separator">/</span>
          <span 
            onClick={() => onNavigate('/' + parts.slice(0, i + 1).join('/'))}
            style={{ cursor: 'pointer' }}
          >
            {part}
          </span>
        </React.Fragment>
      ))}
      <span className="active">●</span>
    </div>
  );
};
