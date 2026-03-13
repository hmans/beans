import { gql } from 'urql';
import { client } from './graphqlClient';

const CONFIG_QUERY = gql`
  query Config {
    projectName
    mainBranch
    agentEnabled
  }
`;

class ConfigStore {
  projectName = $state('');
  mainBranch = $state('main');
  agentEnabled = $state(true);

  async load(): Promise<void> {
    const result = await client.query(CONFIG_QUERY, {}).toPromise();
    if (result.error) {
      console.warn('Failed to load config:', result.error.message);
      return;
    }
    if (result.data) {
      this.projectName = result.data.projectName;
      this.mainBranch = result.data.mainBranch;
      this.agentEnabled = result.data.agentEnabled;
    }
  }
}

export const configStore = new ConfigStore();
