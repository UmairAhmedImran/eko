import type {ReactNode} from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';
import Heading from '@theme/Heading';

import styles from './index.module.css';

function TerminalMockup() {
  return (
    <div className="terminalContainer">
      <div className="terminalHeader">
        <span className="terminalDot dotRed"></span>
        <span className="terminalDot dotYellow"></span>
        <span className="terminalDot dotGreen"></span>
        <span className="terminalTitle">eko bash</span>
      </div>
      <div className="terminalBody">
        <div>
          <span className="terminalPrompt">~$ </span>
          <span className="terminalCmd">eko save --ai</span>
        </div>
        <div className="terminalOutput">
          Snapshot saved: 8c9d1a2f<br />
          AI Summary: Added AI change summary engine and updated SQLite schema migration.
        </div>
        <div style={{marginTop: '0.75rem'}}>
          <span className="terminalPrompt">~$ </span>
          <span className="terminalCmd">eko summary 3b7f2a1e 8c9d1a2f</span>
        </div>
        <div className="terminalHighlight">
          ✦ Snapshot Change Summary [8c9d1a2f]<br />
          Provider: gemini<br />
          Files Changed: 3 (+42 / -12 lines)<br />
          Summary: Added internal/ai provider package and CLI summary command.
        </div>
      </div>
    </div>
  );
}

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <div style={{
          display: 'inline-block',
          padding: '0.35rem 1rem',
          borderRadius: '999px',
          background: 'rgba(255, 255, 255, 0.1)',
          border: '1px solid rgba(255, 255, 255, 0.2)',
          fontSize: '0.85rem',
          fontWeight: 600,
          marginBottom: '1rem',
          letterSpacing: '0.5px'
        }}>
          ✨ NEXT-GEN WORKSPACE TIME MACHINE
        </div>
        <Heading as="h1" className="hero__title">
          {siteConfig.title}
        </Heading>
        <p className="hero__subtitle" style={{maxWidth: '680px', margin: '0 auto 1.5rem'}}>
          {siteConfig.tagline}
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg margin-right--md"
            to="/docs/intro"
            style={{fontWeight: 700}}>
            Get Started →
          </Link>
          <Link
            className="button button--outline button--secondary button--lg"
            to="https://github.com/kavix/eko"
            target="_blank">
            GitHub Repository
          </Link>
        </div>
        <TerminalMockup />
      </div>
    </header>
  );
}

export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`${siteConfig.title} - AI-Powered Snapshot Versioning`}
      description="Lightning-fast, AI-powered directory state versioning CLI written in Go. Capture, compare, and restore directory states concurrently.">
      <HomepageHeader />
      <main style={{paddingTop: '3rem'}}>
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
