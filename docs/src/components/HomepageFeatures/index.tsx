import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  icon: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Concurrent Worker Pool Engine',
    icon: '',
    description: (
      <>
        Capture directory states instantly using Go's worker-pool concurrency model.
        Perform atomic restores with lock-free compare-and-swap (CAS) safety guards.
      </>
    ),
  },
  {
    title: 'AI-Powered Change Summaries',
    icon: '',
    description: (
      <>
        Automatically generate concise, human-readable change summaries using Gemini, OpenAI,
        or local heuristic engines directly when saving or inspecting snapshots.
      </>
    ),
  },
  {
    title: 'Self-Contained SQLite Metadata',
    icon: '',
    description: (
      <>
        All snapshot metadata, file trees, and AI logs are stored locally in a self-contained
        SQLite database inside <code>.eko/</code>. Runs independently of Git.
      </>
    ),
  },
];

function Feature({title, icon, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4', 'margin-bottom--lg')}>
      <div className="featureCard">
        <div className="featureIcon">{icon}</div>
        <Heading as="h3" className="margin-bottom--sm">{title}</Heading>
        <p className="margin-bottom--none" style={{color: 'var(--ifm-color-emphasis-700)', lineHeight: '1.6'}}>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
