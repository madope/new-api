export const buildMainNavLinks = ({
  t,
  docsLink,
  headerNavModules,
  isAuthed,
}) => {
  const defaultModules = {
    home: true,
    console: true,
    pricing: true,
    docs: true,
    about: true,
  };

  const modules = headerNavModules || defaultModules;

  const allLinks = [
    {
      text: t('首页'),
      itemKey: 'home',
      to: '/',
    },
    {
      text: t('控制台'),
      itemKey: 'console',
      to: '/console',
    },
    {
      text: t('模型广场'),
      itemKey: 'pricing',
      to: '/pricing',
    },
    ...(docsLink
      ? [
          {
            text: t('文档'),
            itemKey: 'docs',
            isExternal: true,
            externalLink: docsLink,
          },
        ]
      : []),
    {
      text: t('关于'),
      itemKey: 'about',
      to: '/about',
    },
  ];

  return allLinks.filter((link) => {
    if (link.itemKey === 'docs') {
      return docsLink && modules.docs;
    }

    if (link.itemKey === 'pricing') {
      if (typeof modules.pricing === 'object') {
        if (!modules.pricing.enabled) {
          return false;
        }

        if (modules.pricing.requireAuth && !isAuthed) {
          return false;
        }

        return true;
      }

      return modules.pricing;
    }

    return modules[link.itemKey] === true;
  });
};
