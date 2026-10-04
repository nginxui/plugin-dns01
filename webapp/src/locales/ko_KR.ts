// Korean. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/ko_KR.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'DNS 자격 증명',
  'Select Credential': '자격 증명 선택',
  'Manage': '관리',
  'New credential': '새 자격 증명',
  'Unknown Provider': '알 수 없는 제공자',
  'The previous credential was deleted. Select another one.': '이전에 선택한 자격 증명이 삭제되었습니다. 다른 자격 증명을 선택하십시오.',
  'Advanced options': '고급 옵션',
  'All default': '모두 기본값',
  'Follow CNAME': 'CNAME 따르기',
  'Keep on when the validation record of a domain is delegated to another domain.': '도메인의 검증 레코드가 다른 도메인에 위임된 경우 켜 두십시오.',
  'CNAME following off': 'CNAME 따르기 꺼짐',
  'Check authoritative servers': '권한 있는 서버 확인',
  'When off, waits a fixed 60 seconds before asking for validation.': '끄면 검증을 요청하기 전에 60초 동안 기다립니다.',
  'Authoritative server check off': '권한 있는 서버 확인 꺼짐',
  'Check public resolvers': '공용 리졸버 확인',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': '플러그인 설정에 지정한 재귀 DNS 서버를 통해 레코드가 보이는지 확인합니다.',
  'Public resolver check off': '공용 리졸버 확인 꺼짐',
  // Settings schema
  'Recursive DNS servers': '재귀 DNS 서버',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'DNS 레코드가 보이는지 확인하는 데 사용합니다. 비워 두면 시스템 리졸버를 사용합니다.',
  'Default wait time (seconds)': '기본 대기 시간(초)',
  'How long to wait at most for a record to become visible when the provider does not say.': '제공자가 지정하지 않을 때 레코드가 보이기까지 기다리는 최대 시간입니다.',
}
